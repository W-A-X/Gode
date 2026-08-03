/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

/**
 * Protocol definitions for Go-based file and Git services.
 * These services run as separate mini-services and communicate via WebSocket.
 */

// --- File Service Protocol (port 47811) ---

/** File service default port */
export const FILE_SERVICE_PORT = 47811;

/** File service request */
export interface IFileServiceRequest {
        id: string;
        cmd: string;
        params?: any;
}

/** File service response */
export interface IFileServiceResponse<T = any> {
        id: string;
        success: boolean;
        data?: T;
        error?: string;
}

/** File service event (pushed from server) */
export interface IFileServiceEvent {
        type: string; // 'file.change'
        data: any;
}

// --- File Operation Parameters ---

export interface IReadFileParams {
        path: string;
        encoding?: string;
}

export interface IWriteFileParams {
        path: string;
        content: string;
        encoding?: string;
        create?: boolean;
}

export interface IDeleteParams {
        path: string;
        recursive?: boolean;
}

export interface IMoveParams {
        src: string;
        dst: string;
        overwrite?: boolean;
}

export interface ICopyParams {
        src: string;
        dst: string;
        overwrite?: boolean;
}

export interface IListDirParams {
        path: string;
        recursive?: boolean;
        show_hidden?: boolean;
        extensions?: string[];
}

export interface IMkdirParams {
        path: string;
        recursive?: boolean;
}

export interface IStatParams {
        path: string;
        follow_symlinks?: boolean;
}

export interface IWatchParams {
        path: string;
        recursive?: boolean;
        events?: string[]; // "create", "write", "remove", "rename", "chmod"
}

export interface ISearchParams {
        pattern: string;
        path?: string;
        recursive?: boolean;
        case_insensitive?: boolean;
        extensions?: string[];
        max_results?: number;
}

// --- Response Data Types ---

export interface IFileInfo {
        name: string;
        path: string;
        is_dir: boolean;
        size: number;
        modified_ms: number;
        created_ms: number;
        mode: number;
        is_symlink: boolean;
        extension?: string;
}

export interface IDirEntry extends IFileInfo {
        children?: IDirEntry[];
}

export interface ISearchResult {
        file: string;
        line: number;
        column: number;
        match: string;
        context?: string;
}

export interface IWatchEventData {
        path: string;
        type: string; // "created", "modified", "deleted", "renamed"
        old_path?: string;
}

// --- Git Service Protocol (port 47812) ---

/** Git service default port */
export const GIT_SERVICE_PORT = 47812;

/** Git service request */
export interface IGitServiceRequest {
        id: string;
        cmd: string;
        params?: any;
}

/** Git service response */
export interface IGitServiceResponse<T = any> {
        id: string;
        success: boolean;
        data?: T;
        error?: string;
}

// --- Git Operation Parameters ---

export interface IRepoParams {
        path: string;
}

export interface IStatusParams extends IRepoParams {}

export interface IDiffParams extends IRepoParams {
        file?: string;
        staged?: boolean;
        cached?: boolean;
        commit_hash?: string;
}

export interface ICommitParams extends IRepoParams {
        message: string;
        files?: string[];
        amend?: boolean;
}

export interface IStageParams extends IRepoParams {
        files: string[];
}

export interface IUnstageParams extends IRepoParams {
        files: string[];
}

export interface ICheckoutParams extends IRepoParams {
        ref: string;
        create?: boolean;
        force?: boolean;
}

export interface IBranchListParams extends IRepoParams {
        remote?: boolean;
        all?: boolean;
}

export interface IBranchParams extends IRepoParams {
        name?: string;
        force?: boolean;
}

export interface IPushParams extends IRepoParams {
        remote?: string;
        ref_spec?: string;
        set_upstream?: boolean;
        force?: boolean;
}

export interface IPullParams extends IRepoParams {
        remote?: string;
        branch?: string;
        rebase?: boolean;
}

export interface IFetchParams extends IRepoParams {
        remote?: string;
        prune?: boolean;
        all?: boolean;
}

export interface IMergeParams extends IRepoParams {
        ref: string;
}

export interface IRebaseParams extends IRepoParams {
        ref: string;
}

export interface ILogParams extends IRepoParams {
        limit?: int;
        skip?: int;
        author?: string;
        since?: string;
        until?: string;
        file?: string;
}

export interface IBlameParams extends IRepoParams {
        file: string;
}

export interface IStashParams extends IRepoParams {
        message?: string;
        index?: number;
        action: 'push' | 'pop' | 'drop' | 'list' | 'show';
}

export interface ITagParams extends IRepoParams {
        name?: string;
        message?: string;
        force?: boolean;
        action: 'create' | 'delete' | 'list';
}

export interface ICloneParams {
        url: string;
        path: string;
        branch?: string;
        depth?: number;
        bare?: boolean;
        single_branch?: boolean;
}

export interface IRemoteParams extends IRepoParams {
        name?: string;
        url?: string;
        action: 'add' | 'remove' | 'list' | 'rename';
}

// --- Response Data Types ---

export interface IGitStatus {
        branch: string;
        ahead: number;
        behind: number;
        is_clean: boolean;
        has_untracked: boolean;
        staged?: IFileStatus[];
        unstaged?: IFileStatus[];
        conflicts?: IFileStatus[];
        untracked?: string[];
        head?: ICommitInfo;
        remotes?: IRemoteInfo[];
}

export interface IFileStatus {
        path: string;
        status: string; // "added", "modified", "deleted", "renamed", "copied", "unmerged"
        old_path?: string;
        staged: boolean;
}

export interface IDiffResult {
        path?: string;
        old_path?: string;
        status: string;
        hunks?: IDiffHunk[];
        raw?: string;
}

export interface IDiffHunk {
        old_start: number;
        old_count: number;
        new_start: number;
        new_count: number;
        lines: IDiffLine[];
}

export interface IDiffLine {
        type: string; // "add", "delete", "context"
        content: string;
        old_no?: number;
        new_no?: number;
}

export interface ICommitInfo {
        hash: string;
        short_hash: string;
        author: string;
        email: string;
        date: string;
        message: string;
        parents?: string[];
}

export interface IBranchInfo {
        name: string;
        is_current: boolean;
        is_remote: boolean;
        remote_name?: string;
        commit_hash: string;
        track_status?: string;
        ahead?: number;
        behind?: number;
}

export interface IRemoteInfo {
        name: string;
        url: string;
}

export interface ITagInfo {
        name: string;
        message?: string;
        commit_hash: string;
        is_annotated: boolean;
}

export interface IBlameLine {
        line_number: number;
        hash: string;
        author: string;
        email: string;
        date: string;
        text: string;
}

export interface IBlameResult {
        path: string;
        lines: IBlameLine[];
}

export interface IStashEntry {
        index: number;
        message: string;
}
