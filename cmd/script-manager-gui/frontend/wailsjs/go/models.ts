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
	    exitCode: number;
	    errMsg: string;
	
	    static createFrom(source: any = {}) {
	        return new InlineStatusDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.output = source["output"];
	        this.exitCode = source["exitCode"];
	        this.errMsg = source["errMsg"];
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

