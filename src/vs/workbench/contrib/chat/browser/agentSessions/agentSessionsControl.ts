/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { $ } from '../../../../../base/browser/dom.js';
import { Disposable } from '../../../../../base/common/lifecycle.js';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export interface IAgentSessionsControlOptions {
	overrideStyles?: any;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	filter?: any;
}

export class AgentSessionsControl extends Disposable {
	readonly element: HTMLElement;

	constructor(options: IAgentSessionsControlOptions) {
		super();
		this.element = $('.agentSessionsControl');
	}

	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	setSessions(sessions: any[]): void {
		// stub
	}

	layout(height: number, width: number): void {
		// stub
	}
}
