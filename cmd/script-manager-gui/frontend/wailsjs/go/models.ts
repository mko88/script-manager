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
	    noWait: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ActionDetailDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = source["description"];
	        this.cmd = source["cmd"];
	        this.noWait = source["noWait"];
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
	export class InlineRunResultDTO {
	    output: string;
	    exitCode: number;
	    errMsg: string;
	
	    static createFrom(source: any = {}) {
	        return new InlineRunResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
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
	export class TitlesDTO {
	    items: string;
	    actions: string;
	    details: string;
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new TitlesDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = source["items"];
	        this.actions = source["actions"];
	        this.details = source["details"];
	        this.command = source["command"];
	    }
	}

}

