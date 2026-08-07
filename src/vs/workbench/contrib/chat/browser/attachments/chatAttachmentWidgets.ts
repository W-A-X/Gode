/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { $ } from '../../../../../base/browser/dom.js';
import { Disposable, IDisposable, toDisposable } from '../../../../../base/common/lifecycle.js';
import { localize } from '../../../../../nls.js';
import { RawContextKey } from '../../../../../platform/contextkey/common/contextkey.js';
import { ServicesAccessor } from '../../../../../platform/instantiation/common/instantiation.js';
import { MenuId } from '../../../../../platform/actions/common/actions.js';

export const chatAttachmentResourceContextKey = new RawContextKey<string>('chatAttachmentResource', undefined, { type: 'URI', description: localize('resource', "The full value of the chat attachment resource, including scheme and path") });

export function hookUpSymbolAttachmentDragAndContextMenu(_accessor: ServicesAccessor, _widget: HTMLElement, _parentContextKeyService: unknown, _attachment: unknown, _contextMenuId: MenuId): IDisposable {
	return toDisposable(() => { });
}

abstract class ChatAttachmentWidgetStub extends Disposable {
	readonly element: HTMLElement = $('button', { class: 'chat-attachment-widget-stub' });
}

export class DefaultChatAttachmentWidget extends ChatAttachmentWidgetStub { constructor(..._args: unknown[]) { super(); } }
export class ElementChatAttachmentWidget extends ChatAttachmentWidgetStub { constructor(..._args: unknown[]) { super(); } }
export class FileAttachmentWidget extends ChatAttachmentWidgetStub { constructor(..._args: unknown[]) { super(); } }
export class ImageAttachmentWidget extends ChatAttachmentWidgetStub { constructor(..._args: unknown[]) { super(); } }
export class BrowserViewAttachmentWidget extends ChatAttachmentWidgetStub { constructor(..._args: unknown[]) { super(); } }
export class NotebookCellOutputChatAttachmentWidget extends ChatAttachmentWidgetStub { constructor(..._args: unknown[]) { super(); } }
export class PasteAttachmentWidget extends ChatAttachmentWidgetStub { constructor(..._args: unknown[]) { super(); } }
export class PromptFileAttachmentWidget extends ChatAttachmentWidgetStub { constructor(..._args: unknown[]) { super(); } }
export class PromptTextAttachmentWidget extends ChatAttachmentWidgetStub { constructor(..._args: unknown[]) { super(); } }
export class SCMHistoryItemAttachmentWidget extends ChatAttachmentWidgetStub { constructor(..._args: unknown[]) { super(); } }
export class SCMHistoryItemChangeAttachmentWidget extends ChatAttachmentWidgetStub { constructor(..._args: unknown[]) { super(); } }
export class SCMHistoryItemChangeRangeAttachmentWidget extends ChatAttachmentWidgetStub { constructor(..._args: unknown[]) { super(); } }
export class TerminalCommandAttachmentWidget extends ChatAttachmentWidgetStub { constructor(..._args: unknown[]) { super(); } }
export class ToolSetOrToolItemAttachmentWidget extends ChatAttachmentWidgetStub { constructor(..._args: unknown[]) { super(); } }
