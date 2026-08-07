/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { ChatSessionStatus as AgentSessionStatus, isSessionInProgressStatus } from '../../common/chatSessionsService.js';

export { ChatSessionStatus as AgentSessionStatus, isSessionInProgressStatus } from '../../common/chatSessionsService.js';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function hasValidDiff(changes: any): boolean {
	if (!changes) {
		return false;
	}
	if (changes instanceof Array) {
		return changes.length > 0;
	}
	return changes.files > 0 || changes.insertions > 0 || changes.deletions > 0;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function getAgentChangesSummary(changes: any) {
	if (!changes) {
		return;
	}
	if (!(changes instanceof Array)) {
		return changes;
	}
	let insertions = 0;
	let deletions = 0;
	for (const change of changes) {
		insertions += change.insertions;
		deletions += change.deletions;
	}
	return { files: changes.length, insertions, deletions };
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function getAgentSessionPullRequestContextValue(session: any): 'available' | 'none' {
	return 'none';
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function isSessionInProgressStatusValue(status: any): boolean {
	return isSessionInProgressStatus(status);
}
