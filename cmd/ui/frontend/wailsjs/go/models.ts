export namespace history {
	
	export class AgentError {
	    what: string;
	    why: string;
	    next: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.what = source["what"];
	        this.why = source["why"];
	        this.next = source["next"];
	    }
	}
	export class ToolCall {
	    id: string;
	    action: string;
	    label: string;
	    result: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolCall(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.action = source["action"];
	        this.label = source["label"];
	        this.result = source["result"];
	    }
	}
	export class TokenUsage {
	    input: number;
	    output: number;
	    total: number;
	    contextUsed: number;
	    contextTotal: number;
	    contextSource?: string;
	
	    static createFrom(source: any = {}) {
	        return new TokenUsage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.input = source["input"];
	        this.output = source["output"];
	        this.total = source["total"];
	        this.contextUsed = source["contextUsed"];
	        this.contextTotal = source["contextTotal"];
	        this.contextSource = source["contextSource"];
	    }
	}
	export class Message {
	    id: string;
	    role: string;
	    content: string;
	    time?: string;
	    status: string;
	    elapsedMs?: number;
	    tokens?: number;
	    usage?: TokenUsage;
	    toolCalls?: ToolCall[];
	    error?: AgentError;
	    subagent?: string;
	    warnings?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.role = source["role"];
	        this.content = source["content"];
	        this.time = source["time"];
	        this.status = source["status"];
	        this.elapsedMs = source["elapsedMs"];
	        this.tokens = source["tokens"];
	        this.usage = this.convertValues(source["usage"], TokenUsage);
	        this.toolCalls = this.convertValues(source["toolCalls"], ToolCall);
	        this.error = this.convertValues(source["error"], AgentError);
	        this.subagent = source["subagent"];
	        this.warnings = source["warnings"];
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
	export class Conversation {
	    id: string;
	    title: string;
	    createdAt: string;
	    updatedAt: string;
	    provider: string;
	    model: string;
	    messages: Message[];
	
	    static createFrom(source: any = {}) {
	        return new Conversation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.messages = this.convertValues(source["messages"], Message);
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

export namespace main {
	
	export class contextWindow {
	    total: number;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new contextWindow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.source = source["source"];
	    }
	}

}

export namespace types {
	
	export class Model {
	    id: string;
	    name: string;
	    context_size: number;
	
	    static createFrom(source: any = {}) {
	        return new Model(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.context_size = source["context_size"];
	    }
	}
	export class Provider {
	    name: string;
	    url: string;
	    api_key: string;
	    default_model: number;
	    models: Model[];
	
	    static createFrom(source: any = {}) {
	        return new Provider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	        this.api_key = source["api_key"];
	        this.default_model = source["default_model"];
	        this.models = this.convertValues(source["models"], Model);
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
	export class Config {
	    selected: number;
	    providers: Provider[];
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.selected = source["selected"];
	        this.providers = this.convertValues(source["providers"], Provider);
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

