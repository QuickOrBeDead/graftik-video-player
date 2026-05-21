export namespace data {
	
	export class PlayerDataStore {
	
	
	    static createFrom(source: any = {}) {
	        return new PlayerDataStore(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class PlaylistItemDto {
	    id: string;
	    playlistId: string;
	    path: string;
	    title: string;
	    isPlaying?: boolean;
	    elapsedTime?: number;
	    duration?: number;
	    progressPercent?: number;
	    lastWatched?: number;
	    orderIndex: number;
	
	    static createFrom(source: any = {}) {
	        return new PlaylistItemDto(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.playlistId = source["playlistId"];
	        this.path = source["path"];
	        this.title = source["title"];
	        this.isPlaying = source["isPlaying"];
	        this.elapsedTime = source["elapsedTime"];
	        this.duration = source["duration"];
	        this.progressPercent = source["progressPercent"];
	        this.lastWatched = source["lastWatched"];
	        this.orderIndex = source["orderIndex"];
	    }
	}
	export class PlaylistDto {
	    id: string;
	    name: string;
	    shuffle: boolean;
	    repeat: number;
	    currentItem?: string;
	    currentPlaylistItem?: PlaylistItemDto;
	    currentVolume?: number;
	    items: PlaylistItemDto[];
	
	    static createFrom(source: any = {}) {
	        return new PlaylistDto(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.shuffle = source["shuffle"];
	        this.repeat = source["repeat"];
	        this.currentItem = source["currentItem"];
	        this.currentPlaylistItem = this.convertValues(source["currentPlaylistItem"], PlaylistItemDto);
	        this.currentVolume = source["currentVolume"];
	        this.items = this.convertValues(source["items"], PlaylistItemDto);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class PlaylistListItem {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new PlaylistListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class ThumbnailDataStore {
	
	
	    static createFrom(source: any = {}) {
	        return new ThumbnailDataStore(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class VideoMetadata {
	    duration: number;
	    lastModified: number;
	    fileSize: number;
	    thumbnail: string;
	
	    static createFrom(source: any = {}) {
	        return new VideoMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.duration = source["duration"];
	        this.lastModified = source["lastModified"];
	        this.fileSize = source["fileSize"];
	        this.thumbnail = source["thumbnail"];
	    }
	}

}

export namespace keys {
	
	export class Accelerator {
	    Key: string;
	    Modifiers: string[];
	
	    static createFrom(source: any = {}) {
	        return new Accelerator(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Key = source["Key"];
	        this.Modifiers = source["Modifiers"];
	    }
	}

}

export namespace menu {
	
	export class MenuItem {
	    Label: string;
	    Role: number;
	    Accelerator?: keys.Accelerator;
	    Type: string;
	    Disabled: boolean;
	    Hidden: boolean;
	    Checked: boolean;
	    SubMenu?: Menu;
	
	    static createFrom(source: any = {}) {
	        return new MenuItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Label = source["Label"];
	        this.Role = source["Role"];
	        this.Accelerator = this.convertValues(source["Accelerator"], keys.Accelerator);
	        this.Type = source["Type"];
	        this.Disabled = source["Disabled"];
	        this.Hidden = source["Hidden"];
	        this.Checked = source["Checked"];
	        this.SubMenu = this.convertValues(source["SubMenu"], Menu);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Menu {
	    Items: MenuItem[];
	
	    static createFrom(source: any = {}) {
	        return new Menu(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Items = this.convertValues(source["Items"], MenuItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

