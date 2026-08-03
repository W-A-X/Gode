/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

/**
 * JSON protocol between the VS Code renderer and the gode-engine process.
 * Mirrors gode/editor-go/engine/protocol.go.
 */

/** Fixed local port the gode-engine WebSocket server listens on. */
export const GODE_ENGINE_PORT = 47810;

export interface IPos {
        readonly line: number;
        readonly column: number;
}

export interface IRange {
        readonly start: IPos;
        readonly end: IPos;
}

export interface IInputKey {
        readonly key_type: 'press' | 'release' | 'repeat';
        readonly key: string; // gogpu key name, e.g. 'A', 'Up', 'Enter'
        readonly rune: string;
        readonly shift?: boolean;
        readonly ctrl?: boolean;
        readonly alt?: boolean;
        readonly super?: boolean;
}

export interface IInputMouse {
        readonly mouse_type: 'press' | 'release' | 'move' | 'drag' | 'double_click';
        readonly button: 'left' | 'right' | 'middle';
        readonly x: number;
        readonly y: number;
        readonly shift?: boolean;
        readonly ctrl?: boolean;
        readonly alt?: boolean;
        readonly super?: boolean;
}

export interface IInputWheel {
        readonly dx: number;
        readonly dy: number;
        readonly shift?: boolean;
        readonly ctrl?: boolean;
}

/** A colored, half-open column range [start, end) on a line. Columns are 1-based. */
export interface ITokenSpan {
        readonly start: number;
        readonly end: number;
        /** CSS color string ("#rrggbb" or "rgba(...)") resolved from the VS Code theme color map. */
        readonly color: string;
}

/** The token spans for a single 1-based line. */
export interface ITokenLine {
        readonly line: number;
        readonly spans: readonly ITokenSpan[];
}

/** Tab information for the tab bar protocol. */
export interface ITabInfo {
        /** Display name (typically filename). */
        readonly label: string;
        /** Full path or description shown in tooltip. */
        readonly description: string;
        /** File type icon identifier. */
        readonly icon_name: string;
        /** Unsaved changes indicator. */
        readonly is_dirty: boolean;
        /** Currently focused tab. */
        readonly is_active: boolean;
        /** Pinned/sticky state. */
        readonly is_pinned: boolean;
}

/** Payload for set_tabs command. */
export interface ITabCommand {
        readonly tabs: readonly ITabInfo[];
        readonly active_idx: number;
}

/** Tab bar viewport dimensions. */
export interface ITabViewportCmd {
        readonly width: number;
        readonly height: number;
        readonly scale: number;
}

export interface IGodeCommand {
        readonly cmd: string;

        // open_document
        readonly text?: string;

        // set_viewport
        readonly width?: number;
        readonly height?: number;
        readonly scale?: number;

        // set_glyph_margin_width
        readonly glyph_margin_width?: number;

        // set_breakpoints
        readonly breakpoints?: readonly number[];

        // set_selection
        readonly anchor?: IPos;
        readonly active?: IPos;

        // input
        readonly type?: 'key' | 'mouse' | 'wheel';
        readonly key?: IInputKey;
        readonly mouse?: IInputMouse;
        readonly wheel?: IInputWheel;

        // set_tokens
        readonly tokens?: readonly ITokenLine[];

        // get_content
        readonly id?: number;

        // set_tabs - update tab bar
        readonly tabs?: ITabCommand;

        // tab_viewport - set tab bar rendering dimensions
        readonly tab_viewport?: ITabViewportCmd;
}

export interface IGodeEvent {
        readonly evt: string;

        // frame
        readonly width?: number;
        readonly height?: number;
        readonly data?: Uint8Array | string; // RGBA, row-major; base64 string over WebSocket

        // selection_changed
        readonly anchor?: IPos;
        readonly active?: IPos;

        // edited
        readonly range?: IRange;
        readonly edit_text?: string;

        // get_content response
        readonly id?: number;
        readonly content?: string;

        // tab_frame - rendered tab bar frame data
        readonly tab_width?: number;
        readonly tab_height?: number;
        readonly tab_data?: Uint8Array | string; // RGBA pixels for tab bar

        // tab_selected - user clicked a tab
        readonly tab_selected_idx?: number;

        // tab_close - user clicked close button on a tab
        readonly tab_close_idx?: number;
}
