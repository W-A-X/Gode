/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { Disposable } from '../../../../../base/common/lifecycle.js';
import { CachedListVirtualDelegate } from '../../../../../base/browser/ui/list/list.js';
import { ITreeRenderer, ITreeNode } from '../../../../../base/browser/ui/tree/tree.js';
import { FuzzyScore } from '../../../../../base/common/filters.js';
import { ChatTreeItem } from '../chat.js';

export interface IChatRendererDelegate {
	readonly autoRefocus?: boolean;
}

export interface IChatListItemTemplate {
	readonly container: HTMLElement;
}

export class ChatListItemRenderer extends Disposable implements ITreeRenderer<ChatTreeItem, FuzzyScore, IChatListItemTemplate> {
	static readonly ID = 'chatListItem';

	renderTemplate(container: HTMLElement): IChatListItemTemplate {
		return { container };
	}

	renderElement(_node: ITreeNode<ChatTreeItem, FuzzyScore>, _index: number, _templateData: IChatListItemTemplate): void {
		// stub
	}

	disposeElement(_node: ITreeNode<ChatTreeItem, FuzzyScore>, _index: number, _templateData: IChatListItemTemplate): void {
		// stub
	}

	disposeTemplate(_templateData: IChatListItemTemplate): void {
		// stub
	}
}

export class ChatListDelegate extends CachedListVirtualDelegate<ChatTreeItem> {
	constructor(private readonly defaultElementHeight: number) {
		super();
	}

	protected estimateHeight(_element: ChatTreeItem): number {
		return this.defaultElementHeight;
	}

	getTemplateId(_element: ChatTreeItem): string {
		return ChatListItemRenderer.ID;
	}

	hasDynamicHeight(_element: ChatTreeItem): boolean {
		return true;
	}

	getMeasuredHeight(_element: ChatTreeItem): number | undefined {
		return undefined;
	}

	setDynamicHeight(_element: ChatTreeItem, _height: number): void {
		// stub
	}

	updateDynamicHeight(_element: ChatTreeItem): void {
		// stub
	}

	resetDynamicHeight(_element: ChatTreeItem): void {
		// stub
	}
}
