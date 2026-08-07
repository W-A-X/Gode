/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { URI } from '../../../../../base/common/uri.js';
import { Disposable } from '../../../../../base/common/lifecycle.js';

export class TerminalContext extends Disposable {
	constructor(readonly uri: URI) {
		super();
	}

	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	asAttachment(_widget: any): Promise<any | undefined> {
		return Promise.resolve(undefined);
	}
}
