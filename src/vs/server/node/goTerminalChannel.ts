/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { Emitter, Event } from '../../base/common/event.js';
import { IServerChannel } from '../../base/parts/ipc/common/ipc.js';
import { GoBackendClient } from './goBackendClient.js';
import { RemoteAgentConnectionContext } from '../../platform/remote/common/remoteAgentEnvironment.js';

interface GoProcessInfo {
	id: string;
	processId: number;
	command: string;
	title: string;
}

export class GoTerminalChannel implements IServerChannel<RemoteAgentConnectionContext> {

	constructor(
		private readonly goClient: GoBackendClient,
	) {
	}

	async call(ctx: RemoteAgentConnectionContext, command: string, arg?: any): Promise<any> {
		switch (command) {
			case 'createTerminal': {
				const [id, shellPath, arg_terminal, _cols, _rows, workDir] = arg as [string, string, string[], number, number, string];
				const info = await this.goClient.fetch<GoProcessInfo>('/terminal/create', {
					method: 'POST',
					body: { id, command: shellPath, args: arg_terminal, cols: _cols, rows: _rows, workDir },
				});
				return {
					id: info.id,
					processId: info.processId,
					title: info.title,
					shell: shellPath,
				};
			}
			case 'killTerminal': {
				const id = arg[0] as string;
				await this.goClient.fetch<void>('/terminal/kill', {
					method: 'POST',
					body: { id },
				});
				return undefined;
			}
			case 'sendInput': {
				const id = arg[0] as string;
				const data = arg[1] as string;
				await this.goClient.fetch<void>('/terminal/input', {
					method: 'POST',
					body: { id, data },
				});
				return undefined;
			}
		case 'resizeTerminal': {
			// Go backend handles resize, but it's mostly for PTY support
			return undefined;
		}
			case 'getProcessInfo': {
				const id = arg[0] as string;
				const info = await this.goClient.fetch<GoProcessInfo>(`/terminal/info?id=${encodeURIComponent(id)}`);
				return {
					id: info.id,
					processId: info.processId,
					title: info.title,
				};
			}
			case 'getExitCode': {
				const id = arg[0] as string;
				// Simplified: get from process info
				const info = await this.goClient.fetch<GoProcessInfo>(`/terminal/info?id=${encodeURIComponent(id)}`).catch(() => null);
				if (info) {
					return info.processId; // Simplified
				}
				return -1;
			}
			case 'getDefaultShell': {
				return process.env.SHELL || '/bin/bash';
			}
		}

		throw new Error(`IPC Command ${command} not found in Go terminal channel`);
	}

	listen<T>(ctx: RemoteAgentConnectionContext, event: string, _arg?: any): Event<T> {
		switch (event) {
			case 'onDidCloseTerminal': {
				const emitter = new Emitter<{ id: string; exitCode: number }>();
				return emitter.event as Event<T>;
			}
			case 'onDidSendInput': {
				const emitter = new Emitter<{ id: string; data: string }>();
				return emitter.event as Event<T>;
			}
		}
		throw new Error(`Unknown event ${event}`);
	}
}