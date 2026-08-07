/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { Disposable } from '../../base/common/lifecycle.js';
import { Emitter, Event } from '../../base/common/event.js';
import { ILogService } from '../../platform/log/common/log.js';

export interface GoBackendConfig {
	host: string;
	port: number;
}

export class GoBackendClient extends Disposable {
	private config: GoBackendConfig | null = null;
	private readonly _onDidChangeConnection = this._register(new Emitter<boolean>());
	readonly onDidChangeConnection: Event<boolean> = this._onDidChangeConnection.event;
	private connected = false;

	constructor(private readonly logService: ILogService) {
		super();
		this.discoverServer();
	}

	private async discoverServer(): Promise<void> {
		const portFile = this.getPortFilePath();
		try {
			const fs = await import('fs');
			if (fs.existsSync(portFile)) {
				const port = parseInt(fs.readFileSync(portFile, 'utf-8').trim(), 10);
				if (port > 0) {
					this.config = { host: '127.0.0.1', port };
					this.setConnected(true);
					this.logService.info(`[GoBackendClient] Connected to Go backend at ${this.config.host}:${this.config.port}`);
					return;
				}
			}
		} catch {
			// Port file not found, Go backend may not be running
		}
		this.logService.info('[GoBackendClient] Go backend not found, using TypeScript backend');
	}

	private getPortFilePath(): string {
		const homeDir = process.env.HOME || process.env.USERPROFILE || '.';
		return `${homeDir}/.gode/storage/gode-go-server.port`;
	}

	private setConnected(connected: boolean): void {
		if (this.connected !== connected) {
			this.connected = connected;
			this._onDidChangeConnection.fire(connected);
		}
	}

	isConnected(): boolean {
		return this.connected;
	}

	getConfig(): GoBackendConfig | null {
		return this.config;
	}

	async fetch<T>(path: string, options?: { method?: string; body?: unknown; headers?: Record<string, string> }): Promise<T> {
		if (!this.config) {
			throw new Error('Go backend not connected');
		}

		const url = `http://${this.config.host}:${this.config.port}${path}`;
		const headers = { 'Content-Type': 'application/json', ...options?.headers };
		
		const response = await fetch(url, {
			method: options?.method || 'GET',
			headers,
			body: options?.body ? JSON.stringify(options.body) : undefined,
		});

		if (!response.ok) {
			const errorText = await response.text();
			throw new Error(`Go backend error (${response.status}): ${errorText}`);
		}

		if (response.headers.get('Content-Type')?.includes('application/octet-stream')) {
			const buffer = await response.arrayBuffer();
			return buffer as unknown as T;
		}

		return response.json() as Promise<T>;
	}

	async stream(path: string, options?: { method?: string; body?: unknown; headers?: Record<string, string> }): Promise<ReadableStream<Uint8Array>> {
		if (!this.config) {
			throw new Error('Go backend not connected');
		}

		const url = `http://${this.config.host}:${this.config.port}${path}`;
		const headers = { 'Content-Type': 'application/json', ...options?.headers };

		const response = await fetch(url, {
			method: options?.method || 'GET',
			headers,
			body: options?.body ? JSON.stringify(options.body) : undefined,
		});

		if (!response.ok) {
			throw new Error(`Go backend error (${response.status})`);
		}

		if (!response.body) {
			throw new Error('No response body');
		}

		return response.body;
	}
}