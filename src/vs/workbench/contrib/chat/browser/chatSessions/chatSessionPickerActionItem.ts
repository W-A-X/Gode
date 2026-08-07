/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { Disposable } from '../../../../../base/common/lifecycle.js';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export interface IChatSessionPickerDelegate extends any { }

export class ChatSessionPickerActionItem extends Disposable {
	constructor(..._args: unknown[]) {
		super();
	}
}
