/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { URI } from '../../../../../base/common/uri.js';

export enum ChatSessionPosition {
	Editor = 'editor',
	Sidebar = 'sidebar'
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function getResourceForNewChatSession(options: any): URI {
	const scheme = options?.type === 'editor' ? 'chat-editor' : 'chat';
	return URI.from({ scheme, path: `${Date.now()}.md` });
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function getSessionStatusForModel(model: any): any {
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	return (model as any)?.getSessionStatus?.() ?? undefined;
}
