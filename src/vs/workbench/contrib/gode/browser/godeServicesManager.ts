/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { Disposable } from '../../../../base/common/lifecycle.js';
import { IFileService } from '../../../../platform/files/common/files.js';
import { IInstantiationService } from '../../../../platform/instantiation/common/instantiation.js';
import { IConfigurationService } from '../../../../platform/configuration/common/configurationService.js';
import { GoFileServiceClient } from './goFileServiceClient.js';
import { GoGitServiceClient } from './goGitServiceClient.js';

/**
 * GodeServicesManager manages the Go-based file and Git service clients.
 * It provides a centralized way to access these services and can be
 * used to replace native implementations when gode.services.enabled is true.
 */
export class GodeServicesManager extends Disposable {
        private _fileServiceClient: GoFileServiceClient | null = null;
        private _gitServiceClient: GoGitServiceClient | null = null;
        private _enabled: boolean = false;

        constructor(
                @IInstantiationService private readonly instantiationService: IInstantiationService,
                @IConfigurationService private readonly configurationService: IConfigurationService
        ) {
                super();

                this._enabled = this.configurationService.getValue<boolean>('gode.services.enabled') ?? true;

                if (this._enabled) {
                        this.initializeClients();
                }

                // Listen for configuration changes
                this._register(this.configurationService.onDidChangeConfiguration(e => {
                        if (e.affectsConfiguration('gode.services.enabled')) {
                                const newEnabled = this.configurationService.getValue<boolean>('gode.services.enabled') ?? true;
                                if (newEnabled !== this._enabled) {
                                        this._enabled = newEnabled;
                                        if (this._enabled) {
                                                this.initializeClients();
                                        } else {
                                                this.disposeClients();
                                        }
                                }
                        }
                }));
        }

        private initializeClients(): void {
                console.log('[GodeServices] Initializing Go-based services');
                
                this._fileServiceClient = new GoFileServiceClient();
                this._gitServiceClient = new GoGitServiceClient();
                
                this._register(this._fileServiceClient);
                this._register(this._gitServiceClient);
        }

        private disposeClients(): void {
                console.log('[GodeServices] Disposing Go-based services');
                this._fileServiceClient?.dispose();
                this._gitServiceClient?.dispose();
                this._fileServiceClient = null;
                this._gitServiceClient = null;
        }

        /**
         * Get the file service client.
         */
        get fileService(): GoFileServiceClient | null {
                return this._fileServiceClient;
        }

        /**
         * Get the Git service client.
         */
        get gitService(): GoGitServiceClient | null {
                return this._gitServiceClient;
        }

        /**
         * Check if Go-based services are enabled.
         */
        get isEnabled(): boolean {
                return this._enabled;
        }

        override dispose(): void {
                super.dispose();
                this.disposeClients();
        }
}

// Singleton instance
let _instance: GodeServicesManager | null = null;

/**
 * Get or create the GodeServicesManager singleton.
 */
export function getGodeServicesManager(
        instantiationService: IInstantiationService,
        configurationService: IConfigurationService
): GodeServicesManager {
        if (!_instance || _instance.isDisposed) {
                _instance = instantiationService.createInstance(GodeServicesManager);
        }
        return _instance;
}
