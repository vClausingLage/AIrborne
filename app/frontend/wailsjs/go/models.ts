export namespace gen {
	
	export class Result {
	    game: string;
	    name: string;
	    outputDir: string;
	    mainFile: string;
	    files: string[];
	    notes: string[];
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game = source["game"];
	        this.name = source["name"];
	        this.outputDir = source["outputDir"];
	        this.mainFile = source["mainFile"];
	        this.files = source["files"];
	        this.notes = source["notes"];
	    }
	}

}

export namespace missionfile {
	
	export class Meta {
	    title: plan.Localized;
	    briefing: plan.Localized;
	    author: string;
	    date: string;
	    time: string;
	
	    static createFrom(source: any = {}) {
	        return new Meta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = this.convertValues(source["title"], plan.Localized);
	        this.briefing = this.convertValues(source["briefing"], plan.Localized);
	        this.author = source["author"];
	        this.date = source["date"];
	        this.time = source["time"];
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
	export class Entry {
	    name: string;
	    size: number;
	    text: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.size = source["size"];
	        this.text = source["text"];
	    }
	}
	export class Doc {
	    game: string;
	    path: string;
	    name: string;
	    entries: Entry[];
	    meta: Meta;
	    kneeboards: string[];
	    briefingImages: string[];
	    protected: boolean;
	    dirty: boolean;
	    notes: string[];
	
	    static createFrom(source: any = {}) {
	        return new Doc(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game = source["game"];
	        this.path = source["path"];
	        this.name = source["name"];
	        this.entries = this.convertValues(source["entries"], Entry);
	        this.meta = this.convertValues(source["meta"], Meta);
	        this.kneeboards = source["kneeboards"];
	        this.briefingImages = source["briefingImages"];
	        this.protected = source["protected"];
	        this.dirty = source["dirty"];
	        this.notes = source["notes"];
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

export namespace pipeline {
	
	export class AircraftOption {
	    type: string;
	    label: string;
	    side: string;
	
	    static createFrom(source: any = {}) {
	        return new AircraftOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.label = source["label"];
	        this.side = source["side"];
	    }
	}
	export class ConfigState {
	    base: string;
	    model: string;
	    hasKey: boolean;
	    promptsDir: string;
	    logPath: string;
	    il2MissionsDir: string;
	    il2Editor: string;
	    dcsMissionsDir: string;
	
	    static createFrom(source: any = {}) {
	        return new ConfigState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.base = source["base"];
	        this.model = source["model"];
	        this.hasKey = source["hasKey"];
	        this.promptsDir = source["promptsDir"];
	        this.logPath = source["logPath"];
	        this.il2MissionsDir = source["il2MissionsDir"];
	        this.il2Editor = source["il2Editor"];
	        this.dcsMissionsDir = source["dcsMissionsDir"];
	    }
	}
	export class RoleState {
	    key: string;
	    title: string;
	    hint: string;
	    placeholder: string;
	    input: string;
	
	    static createFrom(source: any = {}) {
	        return new RoleState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.title = source["title"];
	        this.hint = source["hint"];
	        this.placeholder = source["placeholder"];
	        this.input = source["input"];
	    }
	}
	export class State {
	    roles: RoleState[];
	    config: ConfigState;
	    game: string;
	    plan?: plan.MissionPlan;
	    planJson: string;
	    projectPath: string;
	    root: string;
	    issues: string[];
	    output?: gen.Result;
	    prefabs: prefab.Prefab[];
	    aircraft: Record<string, Array<AircraftOption>>;
	    maps: string[];
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.roles = this.convertValues(source["roles"], RoleState);
	        this.config = this.convertValues(source["config"], ConfigState);
	        this.game = source["game"];
	        this.plan = this.convertValues(source["plan"], plan.MissionPlan);
	        this.planJson = source["planJson"];
	        this.projectPath = source["projectPath"];
	        this.root = source["root"];
	        this.issues = source["issues"];
	        this.output = this.convertValues(source["output"], gen.Result);
	        this.prefabs = this.convertValues(source["prefabs"], prefab.Prefab);
	        this.aircraft = this.convertValues(source["aircraft"], Array<AircraftOption>, true);
	        this.maps = source["maps"];
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

export namespace plan {
	
	export class Position {
	    x?: number;
	    z?: number;
	    alt?: number;
	    lat?: number;
	    lon?: number;
	    heading?: number;
	
	    static createFrom(source: any = {}) {
	        return new Position(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.z = source["z"];
	        this.alt = source["alt"];
	        this.lat = source["lat"];
	        this.lon = source["lon"];
	        this.heading = source["heading"];
	    }
	}
	export class Flak {
	    script: string;
	    count: number;
	    position: Position;
	    country?: any;
	    countryName?: string;
	    engageable: boolean;
	    radius?: number;
	
	    static createFrom(source: any = {}) {
	        return new Flak(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.script = source["script"];
	        this.count = source["count"];
	        this.position = this.convertValues(source["position"], Position);
	        this.country = source["country"];
	        this.countryName = source["countryName"];
	        this.engageable = source["engageable"];
	        this.radius = source["radius"];
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
	export class Movement {
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new Movement(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	    }
	}
	export class Start {
	    type: string;
	    x?: number;
	    z?: number;
	    lat?: number;
	    lon?: number;
	    alt: number;
	    heading: number;
	    route?: Position[];
	
	    static createFrom(source: any = {}) {
	        return new Start(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.x = source["x"];
	        this.z = source["z"];
	        this.lat = source["lat"];
	        this.lon = source["lon"];
	        this.alt = source["alt"];
	        this.heading = source["heading"];
	        this.route = this.convertValues(source["route"], Position);
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
	export class Group {
	    name: string;
	    kind?: string;
	    aircraft?: string;
	    script?: string;
	    count?: number;
	    country?: any;
	    countryName?: string;
	    start?: Start;
	    position?: Position;
	    formation?: number[];
	    callsign?: number[];
	    movement?: Movement;
	    route?: Position[];
	    payload?: string;
	    task?: string;
	    notes?: string;
	
	    static createFrom(source: any = {}) {
	        return new Group(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.aircraft = source["aircraft"];
	        this.script = source["script"];
	        this.count = source["count"];
	        this.country = source["country"];
	        this.countryName = source["countryName"];
	        this.start = this.convertValues(source["start"], Start);
	        this.position = this.convertValues(source["position"], Position);
	        this.formation = source["formation"];
	        this.callsign = source["callsign"];
	        this.movement = this.convertValues(source["movement"], Movement);
	        this.route = this.convertValues(source["route"], Position);
	        this.payload = source["payload"];
	        this.task = source["task"];
	        this.notes = source["notes"];
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
	export class Localized {
	    de: string;
	    en: string;
	
	    static createFrom(source: any = {}) {
	        return new Localized(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.de = source["de"];
	        this.en = source["en"];
	    }
	}
	export class Icon {
	    from: Position;
	    to: Position;
	    label: Localized;
	    desc: Localized;
	
	    static createFrom(source: any = {}) {
	        return new Icon(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = this.convertValues(source["from"], Position);
	        this.to = this.convertValues(source["to"], Position);
	        this.label = this.convertValues(source["label"], Localized);
	        this.desc = this.convertValues(source["desc"], Localized);
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
	
	export class Media {
	    briefingImage?: string;
	    kneeboards?: string[];
	    kneeboardBriefing?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Media(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.briefingImage = source["briefingImage"];
	        this.kneeboards = source["kneeboards"];
	        this.kneeboardBriefing = source["kneeboardBriefing"];
	    }
	}
	export class PrefabPlacement {
	    prefab: string;
	    name?: string;
	    side: string;
	    country?: any;
	    position: Position;
	
	    static createFrom(source: any = {}) {
	        return new PrefabPlacement(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prefab = source["prefab"];
	        this.name = source["name"];
	        this.side = source["side"];
	        this.country = source["country"];
	        this.position = this.convertValues(source["position"], Position);
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
	export class Zone {
	    x?: number;
	    z?: number;
	    lat?: number;
	    lon?: number;
	    r: number;
	
	    static createFrom(source: any = {}) {
	        return new Zone(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.z = source["z"];
	        this.lat = source["lat"];
	        this.lon = source["lon"];
	        this.r = source["r"];
	    }
	}
	export class Radio {
	    trigger: string;
	    delay?: number;
	    zone?: Zone;
	    counter?: number;
	    speaker: string;
	    textDe: string;
	    textEn: string;
	
	    static createFrom(source: any = {}) {
	        return new Radio(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.trigger = source["trigger"];
	        this.delay = source["delay"];
	        this.zone = this.convertValues(source["zone"], Zone);
	        this.counter = source["counter"];
	        this.speaker = source["speaker"];
	        this.textDe = source["textDe"];
	        this.textEn = source["textEn"];
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
	export class Objective {
	    title: Localized;
	    desc: Localized;
	    taskType?: number;
	    success?: number;
	    counter: number;
	
	    static createFrom(source: any = {}) {
	        return new Objective(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = this.convertValues(source["title"], Localized);
	        this.desc = this.convertValues(source["desc"], Localized);
	        this.taskType = source["taskType"];
	        this.success = source["success"];
	        this.counter = source["counter"];
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
	export class StaticObject {
	    kind: string;
	    script: string;
	    count: number;
	    country?: any;
	    countryName?: string;
	    positions: Position[];
	
	    static createFrom(source: any = {}) {
	        return new StaticObject(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.script = source["script"];
	        this.count = source["count"];
	        this.country = source["country"];
	        this.countryName = source["countryName"];
	        this.positions = this.convertValues(source["positions"], Position);
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
	export class Weather {
	    cloudLevel: number;
	    cloudHeight: number;
	    precLevel: number;
	    cloudConfig: string;
	    seaState: number;
	    turbulence: number;
	    windLayers: number[][];
	
	    static createFrom(source: any = {}) {
	        return new Weather(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cloudLevel = source["cloudLevel"];
	        this.cloudHeight = source["cloudHeight"];
	        this.precLevel = source["precLevel"];
	        this.cloudConfig = source["cloudConfig"];
	        this.seaState = source["seaState"];
	        this.turbulence = source["turbulence"];
	        this.windLayers = source["windLayers"];
	    }
	}
	export class MissionPlan {
	    game: string;
	    title: Localized;
	    author: string;
	    map: string;
	    date: string;
	    time: string;
	    weather: Weather;
	    playerGroups: Group[];
	    enemyGroups: Group[];
	    friendlyGroups?: Group[];
	    flak: Flak[];
	    statics: StaticObject[];
	    objectives: Objective[];
	    radioQueue: Radio[];
	    briefing: Localized;
	    icons: Icon[];
	    prefabs?: PrefabPlacement[];
	    media?: Media;
	    challenge?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MissionPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game = source["game"];
	        this.title = this.convertValues(source["title"], Localized);
	        this.author = source["author"];
	        this.map = source["map"];
	        this.date = source["date"];
	        this.time = source["time"];
	        this.weather = this.convertValues(source["weather"], Weather);
	        this.playerGroups = this.convertValues(source["playerGroups"], Group);
	        this.enemyGroups = this.convertValues(source["enemyGroups"], Group);
	        this.friendlyGroups = this.convertValues(source["friendlyGroups"], Group);
	        this.flak = this.convertValues(source["flak"], Flak);
	        this.statics = this.convertValues(source["statics"], StaticObject);
	        this.objectives = this.convertValues(source["objectives"], Objective);
	        this.radioQueue = this.convertValues(source["radioQueue"], Radio);
	        this.briefing = this.convertValues(source["briefing"], Localized);
	        this.icons = this.convertValues(source["icons"], Icon);
	        this.prefabs = this.convertValues(source["prefabs"], PrefabPlacement);
	        this.media = this.convertValues(source["media"], Media);
	        this.challenge = source["challenge"];
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

export namespace prefab {
	
	export class Element {
	    name: string;
	    kind: string;
	    type: string;
	    category?: string;
	    shapeName?: string;
	    livery?: string;
	    group?: string;
	    count?: number;
	    spacing?: number;
	    layout?: string;
	    dx: number;
	    dy: number;
	    heading: number;
	    linkTo?: string;
	
	    static createFrom(source: any = {}) {
	        return new Element(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.type = source["type"];
	        this.category = source["category"];
	        this.shapeName = source["shapeName"];
	        this.livery = source["livery"];
	        this.group = source["group"];
	        this.count = source["count"];
	        this.spacing = source["spacing"];
	        this.layout = source["layout"];
	        this.dx = source["dx"];
	        this.dy = source["dy"];
	        this.heading = source["heading"];
	        this.linkTo = source["linkTo"];
	    }
	}
	export class Prefab {
	    id: string;
	    name: string;
	    description: string;
	    game: string;
	    country?: string;
	    tags?: string[];
	    elements: Element[];
	    created?: string;
	
	    static createFrom(source: any = {}) {
	        return new Prefab(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.game = source["game"];
	        this.country = source["country"];
	        this.tags = source["tags"];
	        this.elements = this.convertValues(source["elements"], Element);
	        this.created = source["created"];
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

