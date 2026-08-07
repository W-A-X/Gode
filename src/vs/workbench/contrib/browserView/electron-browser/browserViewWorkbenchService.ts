/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { Emitter, Event } from '../../../../base/common/event.js';
import { Disposable } from '../../../../base/common/lifecycle.js';
import { IBrowserViewWorkbenchService } from '../common/browserView.js';

export const BrowserMaxHistoryEntriesSettingId = 'workbench.browser.maxHistoryEntries';
export const BrowserRemoteProxyEnabledSettingId = 'workbench.browser.enableRemoteProxy';
export const BrowserNewTabPlacementSettingId = 'workbench.browser.newTabPlacement';

export class BrowserViewWorkbenchService extends Disposable implements IBrowserViewWorkbenchService {
	declare readonly _serviceBrand: undefined;

	private readonly _onDidChangeBrowserViews = this._register(new Emitter<void>());
	readonly onDidChangeBrowserViews: Event<void> = this._onDidChangeBrowserViews.event;

	private readonly _onDidChangeSharingAvailable = this._register(new Emitter<boolean>());
	readonly onDidChangeSharingAvailable: Event<boolean> = this._onDidChangeSharingAvailable.event;

	readonly isSharingAvailable = false;

	willUseRemoteProxy(): boolean {
		return false;
	}

	setRemoteProxyInfo(info: any): void {
		// stub
	}

	getKnownBrowserViews(): Map<string, any> {
		return new Map();
	}

	getContextualBrowserViews(): any[] {
		return [];
	}

	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	getBrowserViewContribution(id: string): any {
		return undefined;
	}

	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	createBrowserView(input: any): any {
		return undefined;
	}
}
