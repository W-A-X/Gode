/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { Emitter, Event } from '../../base/common/event.js';
import { IServerChannel } from '../../base/parts/ipc/common/ipc.js';
import { ILogService } from '../../platform/log/common/log.js';
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
		private readonly logService: ILogService,
	) {
	}

	async call(ctx: RemoteAgentConnectionContext, command: string, args: unknown[]): Promise<unknown> {
		switch (command) {
			case 'createTerminal': {
				const [id, shellPath, args_terminal, cols, rows, workDir] = args as [string, string, string[], number, number, string];
				const info = await this.goClient.fetch<GoProcessInfo>('/terminal/create', {
					method: 'POST',
					body: { id, command: shellPath, args: args_terminal, cols, rows, workDir },
				});
				return {
					id: info.id,
					processId: info.processId,
					title: info.title,
					shell: shellPath,
				};
			}
			case 'killTerminal': {
				const id = args[0] as string;
				await this.goClient.fetch<void>('/terminal/kill', {
					method: 'POST',
					body: { id },
				});
				return undefined;
			}
			case 'sendInput': {
				const id = args[0] as string;
				const data = args[1] as string;
				await this.goClient.fetch<void>('/terminal/input', {
					method: 'POST',
					body: { id, data },
				});
				return undefined;
			}
			case 'resizeTerminal': {
				const id = args[0] as string;
				const cols = args[1] as number;
				const rows = args[2] as number;
				// Go backend handles resize, but it's mostly for PTY support
				return undefined;
			}
			case 'getProcessInfo': {
				const id = args[0] as string;
				const info = await this.goClient.fetch<GoProcessInfo>(`/terminal/info?id=${encodeURIComponent(id)}`);
				return {
					id: info.id,
					processId: info.processId,
					title: info.title,
				};
			}
			case 'getExitCode': {
				const id = args[0] as string;
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

	listen(ctx: RemoteAgentConnectionContext, event: string, args: unknown[]): Event<unknown> {
		switch (event) {
			case 'onDidCloseTerminal': {
				const emitter = new Emitter<{ id: string; exitCode: number }>();
				return emitter.event;
			}
			case 'onDidSendInput': {
				const emitter = new Emitter<{ id: string; data: string }>();
				return emitter.event;
			}
		}
		throw new Error(`Unknown event ${event}`);
	}
}