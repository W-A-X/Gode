/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { Emitter, Event } from '../../../../base/common/event.js';
import { Disposable } from '../../../../base/common/lifecycle.js';
import {
        FILE_SERVICE_PORT,
        IFileServiceRequest,
        IFileServiceResponse,
        IFileServiceEvent,
        IFileInfo,
        IDirEntry,
        ISearchResult,
        IWatchEventData,
        IReadFileParams,
        IWriteFileParams,
        IDeleteParams,
        IMoveParams,
        ICopyParams,
        IListDirParams,
        IMkdirParams,
        IStatParams,
        IWatchParams,
        ISearchParams
} from '../common/godeServicesProtocol.js';

/**
 * GoFileServiceClient communicates with the Go-based file service over WebSocket.
 * It provides file system operations that replace native Node.js/Electron file operations.
 */
export class GoFileServiceClient extends Disposable {
        private _ws: WebSocket | null = null;
        private _reconnectTimer: number | null = null;
        private _requestQueue: Map<string, { resolve: (value: any) => void; reject: (error: Error) => void }> = new Map();
        private _requestIdCounter = 0;
        private _disposed = false;

        private readonly _onDidChange = this._register(new Emitter<IWatchEventData>());
        readonly onDidChange: Event<IWatchEventData> = this._onDidChange.event;

        constructor(private readonly _port: number = FILE_SERVICE_PORT) {
                super();
                this.connect();
        }

        private connect(): void {
                if (this._disposed) return;

                try {
                        this._ws = new WebSocket(`ws://127.0.0.1:${this._port}`);
                } catch (err) {
                        console.error('[GoFileService] Failed to connect:', err);
                        this.scheduleReconnect();
                        return;
                }

                this._ws.onopen = () => {
                        console.log('[GoFileService] Connected');
                };

                this._ws.onmessage = (event) => {
                        const data = JSON.parse(event.data);
                        
                        // Check if it's a response or an event
                        if ('cmd' in data || 'success' in data) {
                                const response = data as IFileServiceResponse;
                                const pending = this._requestQueue.get(response.id);
                                if (pending) {
                                        this._requestQueue.delete(response.id);
                                        if (response.success) {
                                                pending.resolve(response.data);
                                        } else {
                                                pending.reject(new Error(response.error || 'Unknown error'));
                                        }
                                }
                        } else if ('type' in data) {
                                const serviceEvent = data as IFileServiceEvent;
                                this._onDidChange.fire(serviceEvent.data as IWatchEventData);
                        }
                };

                this._ws.onclose = () => {
                        this._ws = null;
                        this.scheduleReconnect();
                };

                this._ws.onerror = (err) => {
                        console.error('[GoFileService] WebSocket error:', err);
                        this._ws?.close();
                };
        }

        private scheduleReconnect(): void {
                if (this._disposed || this._reconnectTimer !== null) return;
                this._reconnectTimer = window.setTimeout(() => {
                        this._reconnectTimer = null;
                        this.connect();
                }, 2000);
        }

        private sendRequest<T>(command: string, params?: any): Promise<T> {
                return new Promise<T>((resolve, reject) => {
                        if (!this._ws || this._ws.readyState !== WebSocket.OPEN) {
                                reject(new Error('Not connected to Go file service'));
                                return;
                        }

                        const id = `req_${++this._requestIdCounter}`;
                        const request: IFileServiceRequest = { id, cmd: command, params };
                        
                        this._requestQueue.set(id, { resolve, reject });
                        
                        try {
                                this._ws.send(JSON.stringify(request));
                        } catch (err) {
                                this._requestQueue.delete(id);
                                reject(err);
                        }
                });
        }

        // --- File Read/Write Operations ---

        /**
         * Read file content.
         */
        async readFile(path: string, encoding?: string): Promise<{ content: string; size: number }> {
                const params: IReadFileParams = { path, encoding };
                return this.sendRequest('file.read', params);
        }

        /**
         * Write content to a file.
         */
        async writeFile(path: string, content: string, options?: { create?: boolean }): Promise<void> {
                const params: IWriteFileParams = { path, content, create: options?.create };
                return this.sendRequest('file.write', params);
        }

        /**
         * Append content to a file.
         */
        async appendFile(path: string, content: string): Promise<void> {
                return this.sendRequest('file.append', { path, content });
        }

        // --- File System Operations ---

        /**
         * Delete a file or directory.
         */
        async delete(path: string, recursive?: boolean): Promise<void> {
                const params: IDeleteParams = { path, recursive };
                return this.sendRequest('file.delete', params);
        }

        /**
         * Move/rename a file or directory.
         */
        async move(src: string, dst: string, overwrite?: boolean): Promise<void> {
                const params: IMoveParams = { src, dst, overwrite };
                return this.sendRequest('file.move', params);
        }

        /**
         * Copy a file or directory.
         */
        async copy(src: string, dst: string, overwrite?: boolean): Promise<void> {
                const params: ICopyParams = { src, dst, overwrite };
                return this.sendRequest('file.copy', params);
        }

        /**
         * Create a directory.
         */
        async mkdir(path: string, recursive?: boolean): Promise<void> {
                const params: IMkdirParams = { path, recursive };
                return this.sendRequest('file.mkdir', params);
        }

        // --- File Info Operations ---

        /**
         * Get file/directory metadata.
         */
        async stat(path: string): Promise<IFileInfo> {
                const params: IStatParams = { path };
                return this.sendRequest('file.stat', params);
        }

        /**
         * List directory contents.
         */
        async listDir(path: string, options?: { recursive?: boolean; showHidden?: boolean; extensions?: string[] }): Promise<IDirEntry[]> {
                const params: IListDirParams = {
                        path,
                        recursive: options?.recursive,
                        show_hidden: options?.showHidden,
                        extensions: options?.extensions
                };
                return this.sendRequest('file.list', params);
        }

        /**
         * Check if path exists.
         */
        async exists(path: string): Promise<boolean> {
                const result = await this.sendRequest<{ exists: boolean }>('file.exists', { path });
                return result?.exists ?? false;
        }

        // --- Watch Operations ---

        /**
         * Watch a path for changes.
         */
        async watch(path: string, options?: { recursive?: boolean; events?: string[] }): Promise<void> {
                const params: IWatchParams = { path, ...options };
                return this.sendRequest('file.watch', params);
        }

        /**
         * Stop watching a path.
         */
        async unwatch(path: string): Promise<void> {
                return this.sendRequest('file.unwatch', { path });
        }

        // --- Search Operations ---

        /**
         * Search for pattern in files.
         */
        async search(pattern: string, options?: ISearchParams): Promise<ISearchResult[]> {
                const params: ISearchParams = { pattern, ...options };
                return this.sendRequest('file.search', params);
        }

        // --- Path Utilities ---

        /**
         * Resolve to absolute path.
         */
        async resolvePath(path: string): Promise<string> {
                const result = await this.sendRequest<{ path: string }>('file.resolve', { path });
                return result?.path ?? path;
        }

        /**
         * Get basename of path.
         */
        async basename(path: string): Promise<string> {
                const result = await this.sendRequest<{ name: string }>('file.basename', { path });
                return result?.name ?? path;
        }

        /**
         * Get dirname of path.
         */
        async dirname(path: string): Promise<string> {
                const result = await this.sendRequest<{ dir: string }>('file.dirname', { path });
                return result?.dir ?? path;
        }

        /**
         * Get extension of path.
         */
        async extname(path: string): Promise<string> {
                const result = await this.sendRequest<{ ext: string }>('file.extname', { path });
                return result?.ext ?? '';
        }

        override dispose(): void {
                super.dispose();
                this._disposed = true;
                if (this._reconnectTimer !== null) {
                        window.clearTimeout(this._reconnectTimer);
                        this._reconnectTimer = null;
                }
                this._ws?.close();
                this._ws = null;
        }
}
