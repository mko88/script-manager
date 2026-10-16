export namespace ansi {
	
	export class Span {
	    from: number;
	    to: number;
	    fg?: string;
	    bg?: string;
	    bold?: boolean;
	    dim?: boolean;
	    italic?: boolean;
	    underline?: boolean;
	    strike?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Span(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.fg = source["fg"];
	        this.bg = source["bg"];
	        this.bold = source["bold"];
	        this.dim = source["dim"];
	        this.italic = source["italic"];
	        this.underline = source["underline"];
	        this.strike = source["strike"];
	    }
	}

}

export namespace gui {
	
	export class ActionDTO {
	    index: number;
	    id: string;
	    title: string;
	    groups: string[];
	
	    static createFrom(source: any = {}) {
	        return new ActionDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.id = source["id"];
	        this.title = source["title"];
	        this.groups = source["groups"];
	    }
	}
	export class ActionDetailDTO {
	    description: string;
	    cmd: string;
	    script: string;
	    scriptContent: string;
	    scriptError: string;
	    noWait: boolean;
	    interactive: boolean;
	    language: string;
	    params: scriptparams.Spec;
	    commandLanguage: string;
	
	    static createFrom(source: any = {}) {
	        return new ActionDetailDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = source["description"];
	        this.cmd = source["cmd"];
	        this.script = source["script"];
	        this.scriptContent = source["scriptContent"];
	        this.scriptError = source["scriptError"];
	        this.noWait = source["noWait"];
	        this.interactive = source["interactive"];
	        this.language = source["language"];
	        this.params = this.convertValues(source["params"], scriptparams.Spec);
	        this.commandLanguage = source["commandLanguage"];
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
	export class CommandPreviewDTO {
	    line: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new CommandPreviewDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.line = source["line"];
	        this.error = source["error"];
	    }
	}
	export class DetailsDTO {
	    html: string;
	    copyValues: string[];
	    copyMasked: boolean[];
	    missingFields: string[];
	
	    static createFrom(source: any = {}) {
	        return new DetailsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.html = source["html"];
	        this.copyValues = source["copyValues"];
	        this.copyMasked = source["copyMasked"];
	        this.missingFields = source["missingFields"];
	    }
	}
	export class InlineStatusDTO {
	    running: boolean;
	    output: string;
	    spans: ansi.Span[];
	    exitCode: number;
	    errMsg: string;
	
	    static createFrom(source: any = {}) {
	        return new InlineStatusDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.output = source["output"];
	        this.spans = this.convertValues(source["spans"], ansi.Span);
	        this.exitCode = source["exitCode"];
	        this.errMsg = source["errMsg"];
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
	export class ItemDTO {
	    index: number;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new ItemDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.label = source["label"];
	    }
	}
	export class OpenedConfigDTO {
	    path: string;
	    warning: string;
	
	    static createFrom(source: any = {}) {
	        return new OpenedConfigDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.warning = source["warning"];
	    }
	}
	export class PresetsDTO {
	    presets: presets.Preset[];
	    warning: string;
	
	    static createFrom(source: any = {}) {
	        return new PresetsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.presets = this.convertValues(source["presets"], presets.Preset);
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
	export class RunInputDTO {
	    values: Record<string, any>;
	    args: string;
	
	    static createFrom(source: any = {}) {
	        return new RunInputDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.values = source["values"];
	        this.args = source["args"];
	    }
	}
	export class SecretsStateDTO {
	    configured: boolean;
	    unlocked: boolean;
	    hasLocked: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SecretsStateDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.configured = source["configured"];
	        this.unlocked = source["unlocked"];
	        this.hasLocked = source["hasLocked"];
	    }
	}
	export class UpdateInfoDTO {
	    available: boolean;
	    current: string;
	    latest: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfoDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.current = source["current"];
	        this.latest = source["latest"];
	    }
	}

}

export namespace presets {
	
	export class Preset {
	    name: string;
	    values: Record<string, any>;
	    args: string;
	
	    static createFrom(source: any = {}) {
	        return new Preset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.values = source["values"];
	        this.args = source["args"];
	    }
	}

}

export namespace scriptparams {
	
	export class Param {
	    name: string;
	    flag: string;
	    kind: string;
	    choices: string[];
	    default: string;
	    required: boolean;
	    description: string;
	    repeat: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Param(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.flag = source["flag"];
	        this.kind = source["kind"];
	        this.choices = source["choices"];
	        this.default = source["default"];
	        this.required = source["required"];
	        this.description = source["description"];
	        this.repeat = source["repeat"];
	    }
	}
	export class Spec {
	    language: string;
	    documented: boolean;
	    synopsis: string;
	    params: Param[];
	
	    static createFrom(source: any = {}) {
	        return new Spec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.language = source["language"];
	        this.documented = source["documented"];
	        this.synopsis = source["synopsis"];
	        this.params = this.convertValues(source["params"], Param);
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

export namespace uiprefs {
	
	export class Prefs {
	    fontUi: string;
	    fontMono: string;
	    scalePercent: number;
	    checkForUpdates: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Prefs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fontUi = source["fontUi"];
	        this.fontMono = source["fontMono"];
	        this.scalePercent = source["scalePercent"];
	        this.checkForUpdates = source["checkForUpdates"];
	    }
	}

}

