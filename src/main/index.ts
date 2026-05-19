/* eslint-disable prettier/prettier */
import {
  app,
  shell,
  BrowserWindow,
  dialog,
  Menu,
  MenuItem,
  MenuItemConstructorOptions
} from 'electron'
import path, { join } from 'path'
import { electronApp, optimizer, is } from '@electron-toolkit/utils'
import icon from '../../resources/icon.png?asset'
import { PlayerDataStore } from './data/playerDataStore.js'
import { ipcMainEvents } from './ipcMainEvents.js'
import { ThumbnailDataStore } from './data/thumbnailDataStore.js'
import ffmpeg from 'fluent-ffmpeg'
import ffmpegPath from 'ffmpeg-static'
import { path as ffprobePath } from 'ffprobe-static'

// This function swaps the "asar" path for the "asar.unpacked" path
function getUnpackedPath(binPath: string | null) {
  if (!binPath) return ''
  return !app.isPackaged ? binPath : binPath.replace('app.asar', 'app.asar.unpacked')
}

ffmpeg.setFfmpegPath(getUnpackedPath(ffmpegPath as unknown as string))
ffmpeg.setFfprobePath(getUnpackedPath(ffprobePath))

type WindowOptions = {
  path: string
  parent: BrowserWindow
  minHeight?: number
  minWidth?: number
  maxHeight?: number
  maxWidth?: number
  height?: number
  width?: number
  top?: boolean
  showTaskBar?: boolean
  maximizeable?: boolean
  minimizeable?: boolean
  center?: boolean
}

async function createAppWindow(appOptions?: Partial<WindowOptions>) {
  const {
    parent,
    path,
    minHeight,
    minWidth,
    maxHeight,
    maxWidth,
    height,
    width,
    top,
    showTaskBar,
    minimizeable,
    maximizeable,
    center
  } = appOptions ?? {}

  let route = path
  if (!route) route = '/'
  // Create the browser window.
  const win = new BrowserWindow({
    width: width ?? 800,
    height: height ?? 600,
    minWidth: minWidth ?? 800,
    minHeight: minHeight ?? 480,
    maxWidth,
    maxHeight,
    minimizable: minimizeable === true,
    maximizable: maximizeable === true,
    backgroundColor: '#000000',
    fullscreenable: !maxWidth && !maxWidth,
    parent,
    frame: false,
    modal: parent && top === true,
    skipTaskbar: showTaskBar === false,
    darkTheme: true,
    center: center === true,
    ...(process.platform === 'linux' ? { icon } : {}),
    webPreferences: {
      // Use pluginOptions.nodeIntegration, leave this alone
      // See nklayman.github.io/vue-cli-plugin-electron-builder/guide/security.html#node-integration for more info
      nodeIntegration: process.env.ELECTRON_NODE_INTEGRATION === 'true',
      contextIsolation: true,
      sandbox: false,
      preload: join(__dirname, '../preload/index.mjs')
    }
  })

  win.setMenu(null)

  if (is.dev && process.env['ELECTRON_RENDERER_URL']) {
    // Load the url of the dev server if in development mode
    await win.loadURL(
      `${process.env['ELECTRON_RENDERER_URL'].replace(/\/$/, '')}/#${route.replace(/^\//, '')}`
    )
  } else {
    // Load the index.html when not in development
    await win.loadFile(join(__dirname, '../renderer/index.html'), {
      hash: route.replace(/^\//, '')
    })
  }

  return win
}

let _mainWindow: BrowserWindow
let store: PlayerDataStore
function createMainWindow(): void {
  // Create the browser window.
  const mainWindow = new BrowserWindow({
    width: 1000,
    height: 670,
    minWidth: 800,
    minHeight: 480,
    title: 'video-player',
    show: false,
    center: true,
    ...(process.platform === 'linux' ? { icon } : {}),
    webPreferences: {
      preload: join(__dirname, '../preload/index.mjs'),
      sandbox: false
    }
  })

  _mainWindow = mainWindow

  mainWindow.on('ready-to-show', () => {
    mainWindow.show()
  })

  mainWindow.webContents.setWindowOpenHandler((details) => {
    shell.openExternal(details.url)
    return { action: 'deny' }
  })

  // HMR for renderer base on electron-vite cli.
  // Load the remote URL for development or the local html file for production.
  if (is.dev && process.env['ELECTRON_RENDERER_URL']) {
    mainWindow.loadURL(process.env['ELECTRON_RENDERER_URL'])
  } else {
    mainWindow.loadFile(join(__dirname, '../renderer/index.html'))
  }
}

// This method will be called when Electron has finished
// initialization and is ready to create browser windows.
// Some APIs can only be used after this event occurs.
app.whenReady().then(async () => {
  // Set app user model id for windows
  electronApp.setAppUserModelId('com.electron')

  // Default open or close DevTools by F12 in development
  // and ignore CommandOrControl + R in production.
  // see https://github.com/alex8088/electron-toolkit/tree/master/packages/utils
  app.on('browser-window-created', (_, window) => {
    optimizer.watchWindowShortcuts(window)
  })

  createMainWindow()
  createMenu()

  const migrationsPath = app.isPackaged ? path.join(process.resourcesPath, 'drizzle') : path.join(__dirname, '../../drizzle')

  store = new PlayerDataStore(app.getPath('userData'), migrationsPath)
  await store.AddDefaultPlaylist()

  const thumbnailStore = new ThumbnailDataStore(app.getPath('userData'))

  ipcMainEvents(_mainWindow, store, thumbnailStore)

  app.on('activate', function () {
    // On macOS it's common to re-create a window in the app when the
    // dock icon is clicked and there are no other windows open.
    if (BrowserWindow.getAllWindows().length === 0) {
      createMainWindow()
      createMenu()
    }
  })
})

// Quit when all windows are closed, except on macOS. There, it's common
// for applications and their menu bar to stay active until the user quits
// explicitly with Cmd + Q.
app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit()
  }
})

// In this file you can include the rest of your app"s specific main process
// code. You can also put them in separate files and require them here.
// Create the application menu
function createMenu() {
  const template: Array<MenuItemConstructorOptions | MenuItem> = [
    {
      label: 'File',
      submenu: [
        {
          label: 'Add Video',
          click: async () => {
            const result = await dialog.showOpenDialog(_mainWindow, {
              properties: ['openFile', 'multiSelections'],
              filters: [{ name: 'Videos', extensions: ['mp4', 'mov', 'ogg', 'webm', '3gp'] }]
            })

            if (!result.canceled && result.filePaths.length) {
              _mainWindow.webContents.send('add-playlist-item', store.InitNewPlaylistItems(result.filePaths))
            }
          }
        },
        {
          label: 'New Playlist',
          click: async () => {
            await createAppWindow({
              parent: _mainWindow,
              path: '/add-playlist',
              minWidth: 355,
              minHeight: 125,
              maxWidth: 355,
              maxHeight: 125,
              width: 355,
              height: 125,
              showTaskBar: true,
              top: true
            })
          }
        },
        {
          label: 'Choose Playlist',
          click: async () => {
            await createAppWindow({
              parent: _mainWindow,
              path: '/playlists',
              minWidth: 540,
              minHeight: 340,
              maxWidth: 540,
              maxHeight: 340,
              height: 340,
              width: 540,
              showTaskBar: true,
              top: true,
              center: true
            })
          }
        },
        { type: 'separator' },
        {
          label: 'Exit',
          click: () => {
            app.quit()
          }
        }
      ]
    }
  ]

  const menu = Menu.buildFromTemplate(template)
  Menu.setApplicationMenu(menu)
}
