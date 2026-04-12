export namespace main {
	
	export class CompressOptions {
	    files: string[];
	    outputDir: string;
	    preset: string;
	    targetMB: number;
	    imageQuality: number;
	    audioBitrate: number;
	    pdfQuality: string;
	
	    static createFrom(source: any = {}) {
	        return new CompressOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.files = source["files"];
	        this.outputDir = source["outputDir"];
	        this.preset = source["preset"];
	        this.targetMB = source["targetMB"];
	        this.imageQuality = source["imageQuality"];
	        this.audioBitrate = source["audioBitrate"];
	        this.pdfQuality = source["pdfQuality"];
	    }
	}
	export class CompressResult {
	    inputPath: string;
	    outputPath: string;
	    inputSize: string;
	    outputSize: string;
	    inputRes: string;
	    outputRes: string;
	    fileType: string;
	    error?: string;
	    note?: string;
	
	    static createFrom(source: any = {}) {
	        return new CompressResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inputPath = source["inputPath"];
	        this.outputPath = source["outputPath"];
	        this.inputSize = source["inputSize"];
	        this.outputSize = source["outputSize"];
	        this.inputRes = source["inputRes"];
	        this.outputRes = source["outputRes"];
	        this.fileType = source["fileType"];
	        this.error = source["error"];
	        this.note = source["note"];
	    }
	}
	export class FileInfo {
	    path: string;
	    name: string;
	    sizeMB: number;
	    sizeKB: number;
	    duration: number;
	    width: number;
	    height: number;
	    codec: string;
	    fps: number;
	    isPortrait: boolean;
	    fileType: string;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.sizeMB = source["sizeMB"];
	        this.sizeKB = source["sizeKB"];
	        this.duration = source["duration"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.codec = source["codec"];
	        this.fps = source["fps"];
	        this.isPortrait = source["isPortrait"];
	        this.fileType = source["fileType"];
	    }
	}

}

