/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { CharacterMapping, DomPosition } from '../../common/viewLayout/viewLineRenderer.js';

/**
 * Computes the 1-based column of a character within a DOM node that is a child of
 * a rendered line, given the line's character mapping and the node offset.
 *
 * Moved out of the (removed) DOM view-lines renderer; the DOM-based text rendering
 * is now owned by the Go engine (gode-engine), but sticky scroll, the diff editor
 * and the screen reader render their own lines and still need this utility.
 */
export function getColumnOfNodeOffset(characterMapping: CharacterMapping, spanNode: HTMLElement, offset: number): number {
	const spanNodeTextContentLength = spanNode.textContent.length;

	let spanIndex = -1;
	while (spanNode) {
		spanNode = <HTMLElement>spanNode.previousSibling;
		spanIndex++;
	}

	return characterMapping.getColumn(new DomPosition(spanIndex, offset), spanNodeTextContentLength);
}
