/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { $ } from '../../../../../base/browser/dom.js';
import { Disposable, IDisposable } from '../../../../../base/common/lifecycle.js';

export function canShowAgentsBanner(_chatEntitlementService: unknown): boolean {
	return false;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function createAgentsBanner(options: any, _commandService: any, _telemetryService: any): { disposables: IDisposable; element: HTMLElement } {
	return {
		disposables: Disposable.None,
		element: $('div')
	};
}
