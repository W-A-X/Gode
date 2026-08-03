/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { Disposable } from '../../../../base/common/lifecycle.js';
import {
        GIT_SERVICE_PORT,
        IGitServiceRequest,
        IGitServiceResponse,
        IGitStatus,
        IDiffResult,
        ICommitInfo,
        IBranchInfo,
        IRemoteInfo,
        ITagInfo,
        IBlameResult,
        IStashEntry,
        IStatusParams,
        IDiffParams,
        ICommitParams,
        IStageParams,
        IUnstageParams,
        ICheckoutParams,
        IBranchListParams,
        IBranchParams,
        IPushParams,
        IPullParams,
        IFetchParams,
        IMergeParams,
        IRebaseParams,
        ILogParams,
        IBlameParams,
        IStashParams,
        ITagParams,
        ICloneParams,
        IRemoteParams
} from '../common/godeServicesProtocol.js';

/**
 * GoGitServiceClient communicates with the Go-based Git service over WebSocket.
 * It provides Git operations that replace the vscode.git extension or native git commands.
 */
export class GoGitServiceClient extends Disposable {
        private _ws: WebSocket | null = null;
        private _reconnectTimer: number | null = null;
        private _requestQueue: Map<string, { resolve: (value: any) => void; reject: (error: Error) => void }> = new Map();
        private _requestIdCounter = 0;
        private _disposed = false;

        constructor(private readonly _port: number = GIT_SERVICE_PORT) {
                super();
                this.connect();
        }

        private connect(): void {
                if (this._disposed) return;

                try {
                        this._ws = new WebSocket(`ws://127.0.0.1:${this._port}`);
                } catch (err) {
                        console.error('[GoGitService] Failed to connect:', err);
                        this.scheduleReconnect();
                        return;
                }

                this._ws.onopen = () => {
                        console.log('[GoGitService] Connected');
                };

                this._ws.onmessage = (event) => {
                        const data = JSON.parse(event.data);
                        
                        // Check if it's a response
                        if ('success' in data) {
                                const response = data as IGitServiceResponse;
                                const pending = this._requestQueue.get(response.id);
                                if (pending) {
                                        this._requestQueue.delete(response.id);
                                        if (response.success) {
                                                pending.resolve(response.data);
                                        } else {
                                                pending.reject(new Error(response.error || 'Unknown error'));
                                        }
                                }
                        }
                };

                this._ws.onclose = () => {
                        this._ws = null;
                        this.scheduleReconnect();
                };

                this._ws.onerror = (err) => {
                        console.error('[GoGitService] WebSocket error:', err);
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
                                reject(new Error('Not connected to Go Git service'));
                                return;
                        }

                        const id = `git_req_${++this._requestIdCounter}`;
                        const request: IGitServiceRequest = { id, cmd: command, params };
                        
                        this._requestQueue.set(id, { resolve, reject });
                        
                        try {
                                this._ws.send(JSON.stringify(request));
                        } catch (err) {
                                this._requestQueue.delete(id);
                                reject(err);
                        }
                });
        }

        // --- Repository Operations ---

        /**
         * Get repository status.
         */
        async status(path: string): Promise<IGitStatus> {
                const params: IStatusParams = { path };
                return this.sendRequest('git.status', params);
        }

        /**
         * Clone a repository.
         */
        async clone(url: string, path: string, options?: Partial<ICloneParams>): Promise<void> {
                const params: ICloneParams = { url, path, ...options };
                return this.sendRequest('git.clone', params);
        }

        /**
         * Get diff output.
         */
        async diff(path: string, options?: Partial<IDiffParams>): Promise<IDiffResult[]> {
                const params: IDiffParams = { path, ...options };
                return this.sendRequest('git.diff', params);
        }

        // --- Commit Operations ---

        /**
         * Create a commit.
         */
        async commit(path: string, message: string, options?: Partial<ICommitParams>): Promise<ICommitInfo | undefined> {
                const params: ICommitParams = { path, message, ...options };
                return this.sendRequest('git.commit', params);
        }

        /**
         * Stage files (git add).
         */
        async stage(path: string, files: string[]): Promise<void> {
                const params: IStageParams = { path, files };
                return this.sendRequest('git.stage', params);
        }

        /**
         * Unstage files (git reset).
         */
        async unstage(path: string, files: string[]): Promise<void> {
                const params: IUnstageParams = { path, files };
                return this.sendRequest('git.unstage', params);
        }

        // --- Branch Operations ---

        /**
         * List branches.
         */
        async branchList(path: string, options?: Partial<IBranchListParams>): Promise<IBranchInfo[]> {
                const params: IBranchListParams = { path, ...options };
                return this.sendRequest('git.branch.list', params);
        }

        /**
         * Create a new branch.
         */
        async branchCreate(path: string, name: string): Promise<void> {
                const params: IBranchParams = { path, name };
                return this.sendRequest('git.branch.create', params);
        }

        /**
         * Delete a branch.
         */
        async branchDelete(path: string, name: string, force?: boolean): Promise<void> {
                const params: IBranchParams = { path, name, force };
                return this.sendRequest('git.branch.delete', params);
        }

        /**
         * Rename a branch.
         */
        async branchRename(path: string, oldName: string, newName: string): Promise<void> {
                return this.sendRequest('git.branch.rename', { path, old_name: oldName, new_name: newName });
        }

        // --- Checkout Operations ---

        /**
         * Checkout a branch/commit/tag.
         */
        async checkout(path: string, ref: string, options?: Partial<ICheckoutParams>): Promise<void> {
                const params: ICheckoutParams = { path, ref, ...options };
                return this.sendRequest('git.checkout', params);
        }

        // --- Remote Operations ---

        /**
         * List remotes.
         */
        async remoteList(path: string): Promise<IRemoteInfo[]> {
                return this.sendRequest('git.remote.list', { path });
        }

        /**
         * Add a remote.
         */
        async remoteAdd(path: string, name: string, url: string): Promise<void> {
                const params: IRemoteParams = { path, name, url, action: 'add' };
                return this.sendRequest('git.remote.add', params);
        }

        /**
         * Remove a remote.
         */
        async remoteRemove(path: string, name: string): Promise<void> {
                const params: IRemoteParams = { path, name, action: 'remove' };
                return this.sendRequest('git.remote.remove', params);
        }

        /**
         * Set remote URL.
         */
        async remoteSetURL(path: string, name: string, url: string, push?: boolean): Promise<void> {
                return this.sendRequest('git.remote.set-url', { path, name, url, push });
        }

        // --- Push/Pull/Fetch/Merge ---

        /**
         * Push to remote.
         */
        async push(path: string, options?: Partial<IPushParams>): Promise<void> {
                const params: IPushParams = { path, ...options };
                return this.sendRequest('git.push', params);
        }

        /**
         * Pull from remote.
         */
        async pull(path: string, options?: Partial<IPullParams>): Promise<void> {
                const params: IPullParams = { path, ...options };
                return this.sendRequest('git.pull', params);
        }

        /**
         * Fetch from remote.
         */
        async fetch(path: string, options?: Partial<IFetchParams>): Promise<void> {
                const params: IFetchParams = { path, ...options };
                return this.sendRequest('git.fetch', params);
        }

        /**
         * Merge a branch/commit.
         */
        async merge(path: string, ref: string): Promise<void> {
                const params: IMergeParams = { path, ref };
                return this.sendRequest('git.merge', params);
        }

        /**
         * Rebase onto a branch/commit.
         */
        async rebase(path: string, ref: string): Promise<void> {
                const params: IRebaseParams = { path, ref };
                return this.sendRequest('git.rebase', params);
        }

        // --- Log & Blame ---

        /**
         * Get commit log.
         */
        async log(path: string, options?: Partial<ILogParams>): Promise<ICommitInfo[]> {
                const params: ILogParams = { path, ...options };
                return this.sendRequest('git.log', params);
        }

        /**
         * Get blame for a file.
         */
        async blame(path: string, file: string): Promise<IBlameResult> {
                const params: IBlameParams = { path, file };
                return this.sendRequest('git.blame', params);
        }

        // --- Stash Operations ---

        /**
         * Stash operations (push/pop/drop/list/show).
         */
        async stash(path: string, action: IStashParams['action'], options?: Partial<Omit<IStashParams, 'action'>>): Promise<any> {
                const params: IStashParams = { path, action, ...options };
                return this.sendRequest('git.stash', params);
        }

        // --- Tag Operations ---

        /**
         * List tags.
         */
        async tagList(path: string): Promise<ITagInfo[]> {
                return this.sendRequest('git.tag.list', { path });
        }

        /**
         * Create a tag.
         */
        async tagCreate(path: string, name: string, options?: Partial<Omit<ITagParams, 'action'>>): Promise<void> {
                const params: ITagParams = { path, name, action: 'create', ...options };
                return this.sendRequest('git.tag.create', params);
        }

        /**
         * Delete a tag.
         */
        async tagDelete(path: string, name: string): Promise<void> {
                const params: ITagParams = { path, name, action: 'delete' };
                return this.sendRequest('git.tag.delete', params);
        }

        // --- Reset Operations ---

        /**
         * Reset HEAD.
         */
        async reset(path: string, mode: 'mixed' | 'soft' | 'hard', ref?: string): Promise<void> {
                return this.sendRequest('git.reset', { path, mode, ref });
        }

        /**
         * Hard reset (discard all changes).
         */
        async resetHard(path: string): Promise<void> {
                return this.sendRequest('git.reset.hard', { path });
        }

        // --- Revert & Cherry-pick ---

        /**
         * Revert commits.
         */
        async revert(path: string, refs: string[]): Promise<void> {
                return this.sendRequest('git.revert', { path, refs });
        }

        /**
         * Cherry-pick commits.
         */
        async cherryPick(path: string, refs: string[]): Promise<void> {
                return this.sendCommand('git.cherry-pick', { path, refs });
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
