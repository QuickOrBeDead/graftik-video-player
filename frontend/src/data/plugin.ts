export interface PluginInfo {
  id: string
  name: string
  version: string
  status: string
  menu: MenuEntry[]
  ui?: string
}

export interface MenuEntry {
  label: string
  action: string
}
