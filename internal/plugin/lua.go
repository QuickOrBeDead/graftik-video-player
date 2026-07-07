package plugin

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	graftikLogger "graftik-wails/internal/logger"

	lua "github.com/yuin/gopher-lua"
)

type LuaPlugin struct {
	mu       sync.Mutex
	L        *lua.LState
	Manifest *Manifest
	log      graftikLogger.Logger
}

func LoadLuaPlugin(dir string, log graftikLogger.Logger) (*LuaPlugin, error) {
	if log == nil {
		panic("lua: logger is required")
	}
	log.Debug("lua: loading plugin", "dir", dir)
	cfgPath := filepath.Join(dir, "plugin.json")
	cfgData, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("read plugin.json: %w", err)
	}
	var cfg DiscoveryConfig
	if err := json.Unmarshal(cfgData, &cfg); err != nil {
		return nil, fmt.Errorf("parse plugin.json: %w", err)
	}

	scriptPath := filepath.Join(dir, "main.lua")
	if _, err := os.Stat(scriptPath); err != nil {
		return nil, fmt.Errorf("main.lua not found: %w", err)
	}

	L := lua.NewState()

	p := &LuaPlugin{
		L: L,
		Manifest: &Manifest{
			ID:      cfg.ID,
			Name:    cfg.Name,
			Version: cfg.Version,
		},
		log: log,
	}

	p.registerHostAPI()

	if err := L.DoFile(scriptPath); err != nil {
		L.Close()
		return nil, fmt.Errorf("execute main.lua: %w", err)
	}

	tbl := L.Get(-1)
	table, ok := tbl.(*lua.LTable)
	if !ok {
		L.Close()
		return nil, fmt.Errorf("main.lua must return a table")
	}

	// Store as global for action execution later
	L.SetGlobal("__plugin", table)

	uiVal := table.RawGetString("ui")
	if uiStr, ok := uiVal.(lua.LString); ok {
		p.Manifest.UI = string(uiStr)
	}

	menuVal := table.RawGetString("menuEntries")
	if menuTbl, ok := menuVal.(*lua.LTable); ok {
		menuTbl.ForEach(func(_ lua.LValue, v lua.LValue) {
			if entry, ok := v.(*lua.LTable); ok {
				p.Manifest.MenuEntries = append(p.Manifest.MenuEntries, MenuEntry{
					Label:  lvToString(entry.RawGetString("label")),
					Action: lvToString(entry.RawGetString("action")),
				})
			}
		})
	}

	return p, nil
}

func (p *LuaPlugin) ExecuteAction(action string, args map[string]any) error {
	p.log.Debug("lua: executing action", "plugin", p.Manifest.ID, "action", action, "args", args)
	p.mu.Lock()
	defer p.mu.Unlock()

	pluginTbl := p.L.GetGlobal("__plugin")
	if pluginTbl == lua.LNil {
		return fmt.Errorf("plugin table not found")
	}
	tbl, ok := pluginTbl.(*lua.LTable)
	if !ok {
		return fmt.Errorf("plugin table invalid")
	}

	actionsVal := tbl.RawGetString("actions")
	actionsTbl, ok := actionsVal.(*lua.LTable)
	if !ok {
		return fmt.Errorf("plugin has no actions table")
	}

	fn := actionsTbl.RawGetString(action)
	if fn == lua.LNil {
		return fmt.Errorf("unknown action: %s", action)
	}

	lf, ok := fn.(*lua.LFunction)
	if !ok {
		return fmt.Errorf("action %s is not a function", action)
	}

	p.L.Push(lf)
	if len(args) > 0 {
		argTable := p.L.NewTable()
		for k, v := range args {
			argTable.RawSetString(k, toLuaValue(p.L, v))
		}
		p.L.Push(argTable)
		return p.L.PCall(1, 0, nil)
	}
	return p.L.PCall(0, 0, nil)
}

func (p *LuaPlugin) Close() {
	p.log.Debug("lua: closing plugin", "plugin", p.Manifest.ID)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.L.Close()
}

func (p *LuaPlugin) registerHostAPI() {
	hostTbl := p.L.NewTable()
	hostTbl.RawSetString("exec", p.L.NewFunction(p.hostExec))
	hostTbl.RawSetString("emit", p.L.NewFunction(p.hostEmit))
	hostTbl.RawSetString("addToPlaylist", p.L.NewFunction(p.hostAddToPlaylist))
	p.L.SetGlobal("host", hostTbl)
}

type execLine struct {
	stream string // "stdout" or "stderr"
	text   string
}

func (p *LuaPlugin) hostExec(L *lua.LState) int {
	binary := L.CheckString(1)
	opts := L.CheckTable(2)

	var args []string
	if argsTbl := opts.RawGetString("args"); argsTbl != lua.LNil {
		if tbl, ok := argsTbl.(*lua.LTable); ok {
			tbl.ForEach(func(_ lua.LValue, v lua.LValue) {
				args = append(args, lvToString(v))
			})
		}
	}

	var env []string
	if envTbl := opts.RawGetString("env"); envTbl != lua.LNil {
		if tbl, ok := envTbl.(*lua.LTable); ok {
			tbl.ForEach(func(_ lua.LValue, v lua.LValue) {
				env = append(env, lvToString(v))
			})
		}
	}

	var onStdout, onStderr, onExit *lua.LFunction
	if fn := opts.RawGetString("onStdout"); fn != lua.LNil {
		onStdout, _ = fn.(*lua.LFunction)
	}
	if fn := opts.RawGetString("onStderr"); fn != lua.LNil {
		onStderr, _ = fn.(*lua.LFunction)
	}
	if fn := opts.RawGetString("onExit"); fn != lua.LNil {
		onExit, _ = fn.(*lua.LFunction)
	}

	cmd := exec.Command(binary, args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 1
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 1
	}

	if err := cmd.Start(); err != nil {
		L.Push(lua.LString(err.Error()))
		return 1
	}

	lines := make(chan execLine, 10000)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			lines <- execLine{"stdout", scanner.Text()}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "plugin: stdout scanner error: %v\n", err)
			p.log.Debug("lua: stdout scanner error", "plugin", p.Manifest.ID, "error", err)
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			lines <- execLine{"stderr", scanner.Text()}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "plugin: stderr scanner error: %v\n", err)
			p.log.Debug("lua: stderr scanner error", "plugin", p.Manifest.ID, "error", err)
		}
	}()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	for {
		select {
		case l := <-lines:
			p.callLineCallback(L, l, onStdout, onStderr)
		case err := <-done:
			exitCode := 0
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
			// drain remaining lines
			for {
				select {
				case l := <-lines:
					p.callLineCallback(L, l, onStdout, onStderr)
				default:
					if onExit != nil {
						L.Push(onExit)
						L.Push(lua.LNumber(exitCode))
						L.PCall(1, 0, nil)
					}
					return 0
				}
			}
		}
	}
}

func (p *LuaPlugin) callLineCallback(L *lua.LState, l execLine, onStdout, onStderr *lua.LFunction) {
	p.log.Debug("lua: calling line callback", "plugin", p.Manifest.ID, "stream", l.stream)
	if l.stream == "stdout" && onStdout != nil {
		L.Push(onStdout)
		L.Push(lua.LString(l.text))
		L.PCall(1, 0, nil)
	}
	if l.stream == "stderr" && onStderr != nil {
		L.Push(onStderr)
		L.Push(lua.LString(l.text))
		L.PCall(1, 0, nil)
	}
}

func (p *LuaPlugin) hostEmit(L *lua.LState) int {
	name := L.CheckString(1)
	data := L.Get(2)
	jsonData := toJSON(L, data)
	p.log.Debug("lua: host emit", "plugin", p.Manifest.ID, "event", name)
	eventSinkMu.Lock()
	fn := eventSink
	eventSinkMu.Unlock()
	if fn != nil {
		fn(name, jsonData)
	}
	return 0
}

func (p *LuaPlugin) hostAddToPlaylist(L *lua.LState) int {
	path := L.CheckString(1)
	title := L.OptString(2, "")
	p.log.Debug("lua: host addToPlaylist", "plugin", p.Manifest.ID, "path", path, "title", title)
	addToPlaylistFnMu.Lock()
	fn := addToPlaylistFn
	addToPlaylistFnMu.Unlock()
	if fn != nil {
		fn(path, title)
	}
	return 0
}

var (
	eventSinkMu      sync.Mutex
	eventSink        func(event string, data string)
	addToPlaylistFnMu sync.Mutex
	addToPlaylistFn   func(path, title string)
)

func SetEventSink(fn func(event string, data string)) {
	eventSinkMu.Lock()
	defer eventSinkMu.Unlock()
	eventSink = fn
}

func SetAddToPlaylistFn(fn func(path, title string)) {
	addToPlaylistFnMu.Lock()
	defer addToPlaylistFnMu.Unlock()
	addToPlaylistFn = fn
}

func lvToString(v lua.LValue) string {
	if v == lua.LNil {
		return ""
	}
	return lua.LVAsString(v)
}

func toJSON(L *lua.LState, v lua.LValue) string {
	switch val := v.(type) {
	case lua.LString:
		return string(val)
	case lua.LNumber:
		b, _ := json.Marshal(float64(val))
		return string(b)
	case lua.LBool:
		if bool(val) {
			return "true"
		}
		return "false"
	case *lua.LTable:
		obj := make(map[string]any)
		val.ForEach(func(k, kv lua.LValue) {
			obj[lvToString(k)] = toJSON(L, kv)
		})
		b, _ := json.Marshal(obj)
		return string(b)
	default:
		return ""
	}
}

func toLuaValue(L *lua.LState, v any) lua.LValue {
	switch val := v.(type) {
	case string:
		return lua.LString(val)
	case float64:
		return lua.LNumber(val)
	case bool:
		if val {
			return lua.LTrue
		}
		return lua.LFalse
	case map[string]any:
		tbl := L.NewTable()
		for k, kv := range val {
			tbl.RawSetString(k, toLuaValue(L, kv))
		}
		return tbl
	case nil:
		return lua.LNil
	default:
		return lua.LString(fmt.Sprintf("%v", val))
	}
}
