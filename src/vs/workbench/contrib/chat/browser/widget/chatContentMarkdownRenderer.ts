/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { $ } from '../../../../../base/browser/dom.js';
import { IRenderedMarkdown, IMarkdownRenderer, MarkdownRenderOptions } from '../../../../../platform/markdown/browser/markdownRenderer.js';
import { IMarkdownString } from '../../../../../base/common/htmlContent.js';

export class ChatContentMarkdownRenderer implements IMarkdownRenderer {
	render(_markdown: IMarkdownString, _options?: MarkdownRenderOptions, outElement?: HTMLElement): IRenderedMarkdown {
		const element = outElement ?? $('div');
		element.textContent = _markdown.value ?? '';
		return { element, dispose: () => { } };
	}
}
