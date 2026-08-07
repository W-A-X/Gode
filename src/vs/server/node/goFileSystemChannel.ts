/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { Emitter, Event } from '../../base/common/event.js';
import { IServerChannel } from '../../base/parts/ipc/common/ipc.js';
import { URI, UriComponents } from '../../base/common/uri.js';
import { IURITransformer } from '../../base/common/uriIpc.js';
import { IFileChange, IStat, IFileWriteOptions, IFileDeleteOptions, IFileOverwriteOptions, FileType } from '../../platform/files/common/files.js';
import { VSBuffer } from '../../base/common/buffer.js';
import { GoBackendClient } from './goBackendClient.js';
import { RemoteAgentConnectionContext } from '../../platform/remote/common/remoteAgentEnvironment.js';
import { createURITransformer } from '../../base/common/uriTransformer.js';
import { ILogService } from '../../platform/log/common/log.js';

interface GoStat {
	type: number;
	size: number;
	mtime: number;
	permissions: number;
}

interface GoDirEntry {
	name: string;
	type: number;
}

function goStatToIStat(stat: GoStat): IStat {
	return {
		type: stat.type as FileType,
		size: stat.size,
		mtime: stat.mtime,
		ctime: 0,
		permissions: stat.permissions,
	};
}

export class GoFileSystemChannel implements IServerChannel<RemoteAgentConnectionContext> {

	private readonly uriTransformerCache = new Map<string, IURITransformer>();

	constructor(
		private readonly goClient: GoBackendClient,
		private readonly logService: ILogService,
	) {
	}

	private getUriTransformer(ctx: RemoteAgentConnectionContext): IURITransformer {
		let transformer = this.uriTransformerCache.get(ctx.remoteAuthority);
		if (!transformer) {
			transformer = createURITransformer(ctx.remoteAuthority);
			this.uriTransformerCache.set(ctx.remoteAuthority, transformer);
		}
		return transformer;
	}

	private transformIncoming(uriTransformer: IURITransformer, resource: UriComponents): URI {
		return URI.revive(uriTransformer.transformIncoming(resource));
	}

		async call(ctx: RemoteAgentConnectionContext, command: string, arg?: any): Promise<any> {
		const uriTransformer = this.getUriTransformer(ctx);

		switch (command) {
			case 'stat': {
				const resource = this.transformIncoming(uriTransformer, arg[0] as UriComponents);
				const stat = await this.goClient.fetch<GoStat>(`/fs/stat?path=${encodeURIComponent(resource.fsPath)}`);
				return goStatToIStat(stat);
			}
			case 'realpath': {
				const resource = this.transformIncoming(uriTransformer, arg[0] as UriComponents);
				return this.goClient.fetch<string>(`/fs/realpath?path=${encodeURIComponent(resource.fsPath)}`);
			}
			case 'readdir': {
				const resource = this.transformIncoming(uriTransformer, arg[0] as UriComponents);
				const entries = await this.goClient.fetch<GoDirEntry[]>(`/fs/readdir?path=${encodeURIComponent(resource.fsPath)}`);
				return entries.map(e => [e.name, e.type as FileType] as [string, FileType]);
			}
		case 'open': {
			const resource = this.transformIncoming(uriTransformer, arg[0] as UriComponents);
			const mode = 'w';
				return this.goClient.fetch<number>(`/fs/open`, {
					method: 'POST',
					body: { path: resource.fsPath, mode },
				});
			}
			case 'close': {
				const fd = arg[0] as number;
				return this.goClient.fetch<void>(`/fs/close`, {
					method: 'POST',
					body: { fd },
				});
			}
			case 'read': {
				const fd = arg[0] as number;
				const pos = arg[1] as number;
				const length = arg[2] as number;
				const buffer = await this.goClient.fetch<ArrayBuffer>(`/fs/read?fd=${fd}&pos=${pos}&length=${length}`);
				return [VSBuffer.wrap(new Uint8Array(buffer)), length];
			}
			case 'readFile': {
				const resource = this.transformIncoming(uriTransformer, arg[0] as UriComponents);
				const buffer = await this.goClient.fetch<ArrayBuffer>(`/fs/readfile?path=${encodeURIComponent(resource.fsPath)}`);
				return VSBuffer.wrap(new Uint8Array(buffer));
			}
			case 'write': {
				const fd = arg[0] as number;
				const pos = arg[1] as number;
				const data = arg[2] as VSBuffer;
				const offset = arg[3] as number;
				const length = arg[4] as number;
				return this.goClient.fetch<number>(`/fs/write`, {
					method: 'POST',
					body: { fd, pos, data: Array.from(data.buffer), offset, length },
				});
			}
			case 'writeFile': {
				const resource = this.transformIncoming(uriTransformer, arg[0] as UriComponents);
				const content = arg[1] as VSBuffer;
				const opts = arg[2] as IFileWriteOptions;
				await this.goClient.fetch<void>(`/fs/writefile`, {
					method: 'POST',
					body: { path: resource.fsPath, data: Array.from(content.buffer), mode: opts.append ? 'a' : 'w' },
				});
				return undefined;
			}
			case 'rename': {
				const source = this.transformIncoming(uriTransformer, arg[0] as UriComponents);
				const target = this.transformIncoming(uriTransformer, arg[1] as UriComponents);
				const opts = arg[2] as IFileOverwriteOptions;
				await this.goClient.fetch<void>(`/fs/rename`, {
					method: 'POST',
					body: { source: source.fsPath, target: target.fsPath, overwrite: opts.overwrite },
				});
				return undefined;
			}
			case 'copy': {
				const source = this.transformIncoming(uriTransformer, arg[0] as UriComponents);
				const target = this.transformIncoming(uriTransformer, arg[1] as UriComponents);
				const opts = arg[2] as IFileOverwriteOptions;
				await this.goClient.fetch<void>(`/fs/copy`, {
					method: 'POST',
					body: { source: source.fsPath, target: target.fsPath, overwrite: opts.overwrite },
				});
				return undefined;
			}
			case 'cloneFile': {
				const source = this.transformIncoming(uriTransformer, arg[0] as UriComponents);
				const target = this.transformIncoming(uriTransformer, arg[1] as UriComponents);
				await this.goClient.fetch<void>(`/fs/clonefile`, {
					method: 'POST',
					body: { source: source.fsPath, target: target.fsPath },
				});
				return undefined;
			}
			case 'mkdir': {
				const resource = this.transformIncoming(uriTransformer, arg[0] as UriComponents);
				await this.goClient.fetch<void>(`/fs/mkdir`, {
					method: 'POST',
					body: { path: resource.fsPath },
				});
				return undefined;
			}
			case 'delete': {
				const resource = this.transformIncoming(uriTransformer, arg[0] as UriComponents);
				const opts = arg[1] as IFileDeleteOptions;
				await this.goClient.fetch<void>(`/fs/delete`, {
					method: 'POST',
					body: { path: resource.fsPath, recursive: opts.recursive },
				});
				return undefined;
			}
			case 'watch': {
			const sessionId = arg[0] as string;
			const watchId = arg[1] as number;
			const resource = this.transformIncoming(uriTransformer, arg[2] as UriComponents);
			this.logService.trace(`[GoFileSystemChannel] watch ${resource.fsPath} (session=${sessionId}, id=${watchId})`);
				return watchId;
			}
			case 'unwatch': {
				const sessionId = arg[0] as string;
				const watchId = arg[1] as number;
				this.logService.trace(`[GoFileSystemChannel] unwatch (session=${sessionId}, id=${watchId})`);
				return undefined;
			}
		}

		throw new Error(`IPC Command ${command} not found in Go filesystem channel`);
	}

	listen(ctx: RemoteAgentConnectionContext, event: string, _arg?: any): Event<any> {
		switch (event) {
			case 'fileChange': {
				const emitter = new Emitter<IFileChange[] | string>();
				// File watching is handled via polling in the Go backend
				return emitter.event;
			}
		}
		throw new Error(`Unknown event ${event}`);
	}
}