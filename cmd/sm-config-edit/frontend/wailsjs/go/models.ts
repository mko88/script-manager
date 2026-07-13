export namespace configedit {
	
	export class ActionDTO {
	    id: string;
	    title: string;
	    description: string;
	    cmd: string;
	    script: string;
	    groups: string[];
	    noWait: boolean;
	    interactive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ActionDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.cmd = source["cmd"];
	        this.script = source["script"];
	        this.groups = source["groups"];
	        this.noWait = source["noWait"];
	        this.interactive = source["interactive"];
	    }
	}
	export class ActionGroupDTO {
	    id: string;
	    title: string;
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new ActionGroupDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.color = source["color"];
	    }
	}
	export class ActionPreviewDTO {
	    description: string;
	    cmd: string;
	    script: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new ActionPreviewDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = source["description"];
	        this.cmd = source["cmd"];
	        this.script = source["script"];
	        this.error = source["error"];
	    }
	}
	export class ItemDTO {
	    name: string;
	    display: string;
	    actions: string[];
	    actionGroups: string[];
	    customActions: ActionDTO[];
	    fields: FieldDTO[];
	
	    static createFrom(source: any = {}) {
	        return new ItemDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.display = source["display"];
	        this.actions = source["actions"];
	        this.actionGroups = source["actionGroups"];
	        this.customActions = this.convertValues(source["customActions"], ActionDTO);
	        this.fields = this.convertValues(source["fields"], FieldDTO);
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
	export class FieldDTO {
	    key: string;
	    kind: string;
	    value: string;
	    secret: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FieldDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.kind = source["kind"];
	        this.value = source["value"];
	        this.secret = source["secret"];
	    }
	}
	export class TerminalDTO {
	    mode: string;
	    name: string;
	    argv: string[];
	
	    static createFrom(source: any = {}) {
	        return new TerminalDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.name = source["name"];
	        this.argv = source["argv"];
	    }
	}
	export class DisplayDTO {
	    name: string;
	    list: string;
	    details: string;
	
	    static createFrom(source: any = {}) {
	        return new DisplayDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.list = source["list"];
	        this.details = source["details"];
	    }
	}
	export class ConfigDTO {
	    shell: string[];
	    display: DisplayDTO[];
	    terminal: TerminalDTO;
	    envFields: FieldDTO[];
	    items: ItemDTO[];
	    actionGroups: ActionGroupDTO[];
	    actions: ActionDTO[];
	
	    static createFrom(source: any = {}) {
	        return new ConfigDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.shell = source["shell"];
	        this.display = this.convertValues(source["display"], DisplayDTO);
	        this.terminal = this.convertValues(source["terminal"], TerminalDTO);
	        this.envFields = this.convertValues(source["envFields"], FieldDTO);
	        this.items = this.convertValues(source["items"], ItemDTO);
	        this.actionGroups = this.convertValues(source["actionGroups"], ActionGroupDTO);
	        this.actions = this.convertValues(source["actions"], ActionDTO);
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
	
	
	
	export class PreviewDTO {
	    listLabel: string;
	    detailsHtml: string;
	    missingFields: string[];
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new PreviewDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.listLabel = source["listLabel"];
	        this.detailsHtml = source["detailsHtml"];
	        this.missingFields = source["missingFields"];
	        this.error = source["error"];
	    }
	}
	export class SaveResultDTO {
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	    }
	}
	export class StateDTO {
	    config: ConfigDTO;
	    path: string;
	    warning: string;
	
	    static createFrom(source: any = {}) {
	        return new StateDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config = this.convertValues(source["config"], ConfigDTO);
	        this.path = source["path"];
	        this.warning = source["warning"];
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
	
	export class ValidationIssueDTO {
	    severity: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ValidationIssueDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.severity = source["severity"];
	        this.message = source["message"];
	    }
	}

}

export namespace theme {
	
	export class State {
	    active: string;
	    themes?: Record<string, any>;
	    custom?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.active = source["active"];
	        this.themes = source["themes"];
	        this.custom = source["custom"];
	    }
	}

}

