/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

export const enum SessionConfigKey {
	AutoApprove = 'autoApprove',
	Permissions = 'permissions',
	Isolation = 'isolation',
	Branch = 'branch',
	Mode = 'mode',
	WorktreeBranchPrefix = 'worktreeBranchPrefix',
	WorktreeIncludeFiles = 'worktreeIncludeFiles',
	WorktreeBranchTrack = 'worktreeBranchTrack',
}

export const KNOWN_AUTO_APPROVE_VALUES: ReadonlySet<string> = new Set(['default', 'assisted', 'autoApprove', 'autopilot']);

export const KNOWN_MODE_VALUES: ReadonlySet<string> = new Set(['interactive', 'plan', 'autopilot']);
