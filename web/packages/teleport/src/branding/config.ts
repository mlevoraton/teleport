/**
 * Teleport
 * Copyright (C) 2023  Gravitational, Inc.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

/**
 * White-label branding configuration received from /web/config.js.
 */
export interface BrandingConfig {
  /** Custom product name displayed in navbar/toolbar */
  productName: string;
  /** Root-relative logo URL for the login page */
  logo: string;
  /** Small logo for navbar/compact view */
  logoSmall: string;
  /** Root-relative favicon URL */
  favicon: string;
  /** Custom theme identifier (maps to customThemes map in ThemeProvider) */
  customTheme: string;
}

/** Default branding values (OSS baseline) */
export const DEFAULT_BRANDING: BrandingConfig = {
  productName: 'WarpNavigator',
  logo: '',
  logoSmall: '',
  favicon: '',
  customTheme: '',
};
