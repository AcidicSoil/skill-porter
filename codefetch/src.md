<source_code>
src/cli.js
```
#!/usr/bin/env node

/**
 * Skill Porter CLI
 * Command-line interface for converting between Claude and Gemini formats
 */

import { program } from 'commander';
import chalk from 'chalk';
import { SkillPorter, PLATFORM_TYPES } from './index.js';
import { PRGenerator } from './optional-features/pr-generator.js';
import { ForkSetup } from './optional-features/fork-setup.js';
import fs from 'fs/promises';
import path from 'path';

const porter = new SkillPorter();

// Version from package.json
const packagePath = new URL('../package.json', import.meta.url);
const packageData = JSON.parse(await fs.readFile(packagePath, 'utf8'));

program
  .name('skill-porter')
  .description('Universal tool to convert Claude Code skills to Gemini CLI extensions and vice versa')
  .version(packageData.version);

// Convert command
program
  .command('convert <source-path>')
  .description('Convert a skill or extension between platforms')
  .option('-t, --to <platform>', 'Target platform (claude or gemini)', 'gemini')
  .option('-o, --output <path>', 'Output directory path')
  .option('--no-validate', 'Skip validation after conversion')
  .action(async (sourcePath, options) => {
    try {
      console.log(chalk.blue('\n🔄 Converting skill/extension...\n'));

      const result = await porter.convert(
        path.resolve(sourcePath),
        options.to,
        {
          outputPath: options.output ? path.resolve(options.output) : undefined,
          validate: options.validate !== false
        }
      );

      if (result.success) {
        console.log(chalk.green('✓ Conversion successful!\n'));

        if (result.files && result.files.length > 0) {
          console.log(chalk.bold('Generated files:'));
          result.files.forEach(file => {
            console.log(chalk.gray(`  - ${file}`));
          });
          console.log();
        }

        if (result.validation) {
          if (result.validation.valid) {
            console.log(chalk.green('✓ Validation passed\n'));
          } else {
            console.log(chalk.yellow('⚠ Validation warnings:\n'));
            result.validation.errors.forEach(error => {
              console.log(chalk.yellow(`  - ${error}`));
            });
            console.log();
          }

          if (result.validation.warnings && result.validation.warnings.length > 0) {
            console.log(chalk.yellow('Warnings:'));
            result.validation.warnings.forEach(warning => {
              console.log(chalk.yellow(`  - ${warning}`));
            });
            console.log();
          }
        }

        // Installation instructions
        const targetPlatform = options.to;
        console.log(chalk.bold('Next steps:'));
        if (targetPlatform === PLATFORM_TYPES.GEMINI) {
          console.log(chalk.gray(`  gemini extensions install ${options.output || sourcePath}`));
        } else {
          console.log(chalk.gray(`  cp -r ${options.output || sourcePath} ~/.claude/skills/`));
        }
        console.log();
      } else {
        console.log(chalk.red('✗ Conversion failed\n'));
        if (result.errors && result.errors.length > 0) {
          console.log(chalk.red('Errors:'));
          result.errors.forEach(error => {
            console.log(chalk.red(`  - ${error}`));
          });
          console.log();
        }
        process.exit(1);
      }
    } catch (error) {
      console.error(chalk.red(`\n✗ Error: ${error.message}\n`));
      process.exit(1);
    }
  });

// Analyze command
program
  .command('analyze <path>')
  .description('Analyze a directory to detect platform type')
  .action(async (dirPath) => {
    try {
      console.log(chalk.blue('\n🔍 Analyzing directory...\n'));

      const detection = await porter.analyze(path.resolve(dirPath));

      console.log(chalk.bold('Detection Results:'));
      console.log(chalk.gray(`  Platform: ${chalk.white(detection.platform)}`));
      console.log(chalk.gray(`  Confidence: ${chalk.white(detection.confidence)}\n`));

      if (detection.files.claude.length > 0) {
        console.log(chalk.bold('Claude files found:'));
        detection.files.claude.forEach(file => {
          const status = file.valid ? chalk.green('✓') : chalk.red('✗');
          const issue = file.issue ? chalk.gray(` (${file.issue})`) : '';
          console.log(`  ${status} ${file.file}${issue}`);
        });
        console.log();
      }

      if (detection.files.gemini.length > 0) {
        console.log(chalk.bold('Gemini files found:'));
        detection.files.gemini.forEach(file => {
          const status = file.valid ? chalk.green('✓') : chalk.red('✗');
          const issue = file.issue ? chalk.gray(` (${file.issue})`) : '';
          console.log(`  ${status} ${file.file}${issue}`);
        });
        console.log();
      }

      if (detection.files.shared.length > 0) {
        console.log(chalk.bold('Shared files found:'));
        detection.files.shared.forEach(file => {
          console.log(chalk.gray(`  - ${file.file}`));
        });
        console.log();
      }

      if (detection.metadata.claude || detection.metadata.gemini) {
        console.log(chalk.bold('Metadata:'));
        if (detection.metadata.claude) {
          console.log(chalk.gray(`  Name: ${detection.metadata.claude.name || 'N/A'}`));
          console.log(chalk.gray(`  Description: ${detection.metadata.claude.description || 'N/A'}`));
        }
        if (detection.metadata.gemini) {
          console.log(chalk.gray(`  Name: ${detection.metadata.gemini.name || 'N/A'}`));
          console.log(chalk.gray(`  Version: ${detection.metadata.gemini.version || 'N/A'}`));
        }
        console.log();
      }
    } catch (error) {
      console.error(chalk.red(`\n✗ Error: ${error.message}\n`));
      process.exit(1);
    }
  });

// Validate command
program
  .command('validate <path>')
  .description('Validate a skill or extension')
  .option('-p, --platform <type>', 'Platform type (claude, gemini, or universal)')
  .action(async (dirPath, options) => {
    try {
      console.log(chalk.blue('\n✓ Validating...\n'));

      const validation = await porter.validate(
        path.resolve(dirPath),
        options.platform
      );

      if (validation.valid) {
        console.log(chalk.green('✓ Validation passed!\n'));
      } else {
        console.log(chalk.red('✗ Validation failed\n'));
      }

      if (validation.errors && validation.errors.length > 0) {
        console.log(chalk.red('Errors:'));
        validation.errors.forEach(error => {
          console.log(chalk.red(`  - ${error}`));
        });
        console.log();
      }

      if (validation.warnings && validation.warnings.length > 0) {
        console.log(chalk.yellow('Warnings:'));
        validation.warnings.forEach(warning => {
          console.log(chalk.yellow(`  - ${warning}`));
        });
        console.log();
      }

      if (!validation.valid) {
        process.exit(1);
      }
    } catch (error) {
      console.error(chalk.red(`\n✗ Error: ${error.message}\n`));
      process.exit(1);
    }
  });

// Make universal command
program
  .command('universal <source-path>')
  .description('Make a skill/extension work on both platforms')
  .option('-o, --output <path>', 'Output directory path')
  .action(async (sourcePath, options) => {
    try {
      console.log(chalk.blue('\n🌐 Creating universal skill/extension...\n'));

      const result = await porter.makeUniversal(
        path.resolve(sourcePath),
        {
          outputPath: options.output ? path.resolve(options.output) : undefined
        }
      );

      if (result.success) {
        console.log(chalk.green('✓ Successfully created universal skill/extension!\n'));
        console.log(chalk.gray('Your skill/extension now works with both Claude Code and Gemini CLI.\n'));
      } else {
        console.log(chalk.red('✗ Failed to create universal skill/extension\n'));
        if (result.errors && result.errors.length > 0) {
          result.errors.forEach(error => {
            console.log(chalk.red(`  - ${error}`));
          });
          console.log();
        }
        process.exit(1);
      }
    } catch (error) {
      console.error(chalk.red(`\n✗ Error: ${error.message}\n`));
      process.exit(1);
    }
  });

// Create PR command
program
  .command('create-pr <source-path>')
  .description('Create a pull request to add dual-platform support')
  .option('-t, --to <platform>', 'Target platform to add (claude or gemini)', 'gemini')
  .option('-b, --base <branch>', 'Base branch for PR', 'main')
  .option('-r, --remote <name>', 'Git remote name', 'origin')
  .option('--draft', 'Create as draft PR')
  .action(async (sourcePath, options) => {
    try {
      console.log(chalk.blue('\n📝 Creating pull request...\n'));

      // First, convert if not already done
      const result = await porter.convert(
        path.resolve(sourcePath),
        options.to,
        { validate: true }
      );

      if (!result.success) {
        console.log(chalk.red('✗ Conversion failed\n'));
        result.errors.forEach(error => console.log(chalk.red(`  - ${error}`)));
        process.exit(1);
      }

      console.log(chalk.green('✓ Conversion completed\n'));

      // Generate PR
      const prGen = new PRGenerator(path.resolve(sourcePath));
      const prResult = await prGen.generate({
        targetPlatform: options.to,
        remote: options.remote,
        baseBranch: options.base,
        draft: options.draft
      });

      if (prResult.success) {
        console.log(chalk.green('✓ Pull request created!\n'));
        console.log(chalk.bold('PR URL:'));
        console.log(chalk.cyan(`  ${prResult.prUrl}\n`));
        console.log(chalk.gray(`Branch: ${prResult.branch}\n`));
      } else {
        console.log(chalk.red('✗ Failed to create pull request\n'));
        prResult.errors.forEach(error => {
          console.log(chalk.red(`  - ${error}`));
        });
        process.exit(1);
      }
    } catch (error) {
      console.error(chalk.red(`\n✗ Error: ${error.message}\n`));
      process.exit(1);
    }
  });

// Fork setup command
program
  .command('fork <source-path>')
  .description('Create a fork with dual-platform setup')
  .option('-l, --location <path>', 'Fork location directory', '.')
  .option('-u, --url <url>', 'Repository URL to clone (optional)')
  .action(async (sourcePath, options) => {
    try {
      console.log(chalk.blue('\n🍴 Setting up fork with dual-platform support...\n'));

      const forkSetup = new ForkSetup(path.resolve(sourcePath));
      const result = await forkSetup.setup({
        forkLocation: path.resolve(options.location),
        repoUrl: options.url
      });

      if (result.success) {
        console.log(chalk.green('✓ Fork created successfully!\n'));
        console.log(chalk.bold('Fork location:'));
        console.log(chalk.cyan(`  ${result.forkPath}\n`));

        console.log(chalk.bold('Installations:'));
        console.log(chalk.gray(`  Claude Code: ${result.installations.claude || 'N/A'}`));
        console.log(chalk.gray(`  Gemini CLI:  ${result.installations.gemini || 'N/A'}\n`));

        console.log(chalk.bold('Next steps:'));
        console.log(chalk.gray('  1. Navigate to fork: cd ' + result.forkPath));
        console.log(chalk.gray('  2. For Gemini: ' + result.installations.gemini));
        console.log(chalk.gray('  3. Test on both platforms\n'));
      } else {
        console.log(chalk.red('✗ Fork setup failed\n'));
        result.errors.forEach(error => {
          console.log(chalk.red(`  - ${error}`));
        });
        process.exit(1);
      }
    } catch (error) {
      console.error(chalk.red(`\n✗ Error: ${error.message}\n`));
      process.exit(1);
    }
  });

program.parse();
```

src/index.js
```
/**
 * Skill Porter - Main Module
 * Universal tool to convert Claude Code skills to Gemini CLI extensions and vice versa
 */

import { PlatformDetector, PLATFORM_TYPES } from './analyzers/detector.js';
import { Validator } from './analyzers/validator.js';
import { ClaudeToGeminiConverter } from './converters/claude-to-gemini.js';
import { GeminiToClaudeConverter } from './converters/gemini-to-claude.js';

export class SkillPorter {
  constructor() {
    this.detector = new PlatformDetector();
    this.validator = new Validator();
  }

  /**
   * Analyze a skill/extension directory
   * @param {string} dirPath - Path to the directory to analyze
   * @returns {Promise<object>} Detection results
   */
  async analyze(dirPath) {
    return await this.detector.detect(dirPath);
  }

  /**
   * Convert a skill/extension
   * @param {string} sourcePath - Source directory path
   * @param {string} targetPlatform - Target platform ('claude' or 'gemini')
   * @param {object} options - Conversion options
   * @returns {Promise<object>} Conversion results
   */
  async convert(sourcePath, targetPlatform, options = {}) {
    const { outputPath = sourcePath, validate = true } = options;

    // Step 1: Detect source platform
    const detection = await this.detector.detect(sourcePath);

    if (detection.platform === PLATFORM_TYPES.UNKNOWN) {
      throw new Error('Unable to detect platform type. Ensure directory contains valid skill/extension files.');
    }

    // Step 2: Check if conversion is needed
    if (detection.platform === PLATFORM_TYPES.UNIVERSAL) {
      return {
        success: true,
        message: 'Already a universal skill/extension - no conversion needed',
        platform: PLATFORM_TYPES.UNIVERSAL
      };
    }

    if (detection.platform === targetPlatform) {
      return {
        success: true,
        message: `Already a ${targetPlatform} ${targetPlatform === 'claude' ? 'skill' : 'extension'} - no conversion needed`,
        platform: detection.platform
      };
    }

    // Step 3: Perform conversion
    let converter;
    let result;

    if (targetPlatform === PLATFORM_TYPES.GEMINI) {
      converter = new ClaudeToGeminiConverter(sourcePath, outputPath);
      result = await converter.convert();
    } else if (targetPlatform === PLATFORM_TYPES.CLAUDE) {
      converter = new GeminiToClaudeConverter(sourcePath, outputPath);
      result = await converter.convert();
    } else {
      throw new Error(`Invalid target platform: ${targetPlatform}. Must be 'claude' or 'gemini'`);
    }

    // Step 4: Validate if requested
    if (validate && result.success) {
      const validation = await this.validator.validate(outputPath, targetPlatform);
      result.validation = validation;

      if (!validation.valid) {
        result.success = false;
        result.errors = result.errors || [];
        result.errors.push('Validation failed', ...validation.errors);
      }
    }

    return result;
  }

  /**
   * Validate a skill/extension
   * @param {string} dirPath - Directory path to validate
   * @param {string} platform - Platform type ('claude', 'gemini', or 'universal')
   * @returns {Promise<object>} Validation results
   */
  async validate(dirPath, platform = null) {
    // Auto-detect platform if not specified
    if (!platform) {
      const detection = await this.detector.detect(dirPath);
      platform = detection.platform;
    }

    return await this.validator.validate(dirPath, platform);
  }

  /**
   * Create a universal skill/extension (both platforms)
   * @param {string} sourcePath - Source directory path
   * @param {object} options - Creation options
   * @returns {Promise<object>} Creation results
   */
  async makeUniversal(sourcePath, options = {}) {
    const { outputPath = sourcePath } = options;

    // Detect current platform
    const detection = await this.detector.detect(sourcePath);

    if (detection.platform === PLATFORM_TYPES.UNIVERSAL) {
      return {
        success: true,
        message: 'Already a universal skill/extension',
        platform: PLATFORM_TYPES.UNIVERSAL
      };
    }

    if (detection.platform === PLATFORM_TYPES.UNKNOWN) {
      throw new Error('Unable to detect platform type');
    }

    // Convert to the other platform while keeping the original
    const targetPlatform = detection.platform === PLATFORM_TYPES.CLAUDE ?
      PLATFORM_TYPES.GEMINI : PLATFORM_TYPES.CLAUDE;

    const result = await this.convert(sourcePath, targetPlatform, {
      outputPath,
      validate: true
    });

    if (result.success) {
      result.platform = PLATFORM_TYPES.UNIVERSAL;
      result.message = 'Successfully created universal skill/extension';
    }

    return result;
  }
}

// Export main class and constants
export { PLATFORM_TYPES } from './analyzers/detector.js';
export default SkillPorter;
```

src/analyzers/detector.js
```
/**
 * Platform Detection
 * Analyzes a directory to determine if it's a Claude skill, Gemini extension, or universal
 */

import fs from 'fs/promises';
import path from 'path';

export const PLATFORM_TYPES = {
  CLAUDE: 'claude',
  GEMINI: 'gemini',
  UNIVERSAL: 'universal',
  UNKNOWN: 'unknown'
};

export class PlatformDetector {
  /**
   * Detect the platform type of a skill/extension directory
   * @param {string} dirPath - Path to the directory to analyze
   * @returns {Promise<{platform: string, files: object, confidence: string}>}
   */
  async detect(dirPath) {
    const detection = {
      platform: PLATFORM_TYPES.UNKNOWN,
      files: {
        claude: [],
        gemini: [],
        shared: []
      },
      confidence: 'low',
      metadata: {}
    };

    try {
      const exists = await this._checkDirectoryExists(dirPath);
      if (!exists) {
        throw new Error(`Directory not found: ${dirPath}`);
      }

      // Check for Claude-specific files
      const claudeFiles = await this._detectClaudeFiles(dirPath);
      detection.files.claude = claudeFiles;

      // Check for Gemini-specific files
      const geminiFiles = await this._detectGeminiFiles(dirPath);
      detection.files.gemini = geminiFiles;

      // Check for shared files
      const sharedFiles = await this._detectSharedFiles(dirPath);
      detection.files.shared = sharedFiles;

      // Determine platform type
      const hasClaude = claudeFiles.length > 0;
      const hasGemini = geminiFiles.length > 0;

      if (hasClaude && hasGemini) {
        detection.platform = PLATFORM_TYPES.UNIVERSAL;
        detection.confidence = 'high';
      } else if (hasClaude) {
        detection.platform = PLATFORM_TYPES.CLAUDE;
        detection.confidence = 'high';
      } else if (hasGemini) {
        detection.platform = PLATFORM_TYPES.GEMINI;
        detection.confidence = 'high';
      } else {
        detection.platform = PLATFORM_TYPES.UNKNOWN;
        detection.confidence = 'low';
      }

      // Extract metadata
      detection.metadata = await this._extractMetadata(dirPath, detection.platform);

      return detection;
    } catch (error) {
      throw new Error(`Detection failed: ${error.message}`);
    }
  }

  /**
   * Check if directory exists
   */
  async _checkDirectoryExists(dirPath) {
    try {
      const stats = await fs.stat(dirPath);
      return stats.isDirectory();
    } catch {
      return false;
    }
  }

  /**
   * Detect Claude-specific files
   */
  async _detectClaudeFiles(dirPath) {
    const claudeFiles = [];

    // Check for SKILL.md
    const skillPath = path.join(dirPath, 'SKILL.md');
    if (await this._fileExists(skillPath)) {
      const hasValidFrontmatter = await this._hasYAMLFrontmatter(skillPath);
      if (hasValidFrontmatter) {
        claudeFiles.push({ file: 'SKILL.md', type: 'entry', valid: true });
      } else {
        claudeFiles.push({ file: 'SKILL.md', type: 'entry', valid: false, issue: 'Missing or invalid YAML frontmatter' });
      }
    }

    // Check for .claude-plugin/marketplace.json
    const marketplacePath = path.join(dirPath, '.claude-plugin', 'marketplace.json');
    if (await this._fileExists(marketplacePath)) {
      const isValidJSON = await this._isValidJSON(marketplacePath);
      if (isValidJSON) {
        claudeFiles.push({ file: '.claude-plugin/marketplace.json', type: 'manifest', valid: true });
      } else {
        claudeFiles.push({ file: '.claude-plugin/marketplace.json', type: 'manifest', valid: false, issue: 'Invalid JSON' });
      }
    }

    return claudeFiles;
  }

  /**
   * Detect Gemini-specific files
   */
  async _detectGeminiFiles(dirPath) {
    const geminiFiles = [];

    // Check for gemini-extension.json
    const manifestPath = path.join(dirPath, 'gemini-extension.json');
    if (await this._fileExists(manifestPath)) {
      const isValidJSON = await this._isValidJSON(manifestPath);
      if (isValidJSON) {
        geminiFiles.push({ file: 'gemini-extension.json', type: 'manifest', valid: true });
      } else {
        geminiFiles.push({ file: 'gemini-extension.json', type: 'manifest', valid: false, issue: 'Invalid JSON' });
      }
    }

    // Check for GEMINI.md
    const geminiMdPath = path.join(dirPath, 'GEMINI.md');
    if (await this._fileExists(geminiMdPath)) {
      geminiFiles.push({ file: 'GEMINI.md', type: 'context', valid: true });
    }

    return geminiFiles;
  }

  /**
   * Detect shared files (common to both platforms)
   */
  async _detectSharedFiles(dirPath) {
    const sharedFiles = [];

    // Check for package.json
    const packagePath = path.join(dirPath, 'package.json');
    if (await this._fileExists(packagePath)) {
      sharedFiles.push({ file: 'package.json', type: 'dependency' });
    }

    // Check for shared directory
    const sharedDirPath = path.join(dirPath, 'shared');
    if (await this._checkDirectoryExists(sharedDirPath)) {
      sharedFiles.push({ file: 'shared/', type: 'directory' });
    }

    // Check for MCP server directory
    const mcpServerPath = path.join(dirPath, 'mcp-server');
    if (await this._checkDirectoryExists(mcpServerPath)) {
      sharedFiles.push({ file: 'mcp-server/', type: 'directory' });
    }

    return sharedFiles;
  }

  /**
   * Extract metadata from files
   */
  async _extractMetadata(dirPath, platform) {
    const metadata = {};

    if (platform === PLATFORM_TYPES.CLAUDE || platform === PLATFORM_TYPES.UNIVERSAL) {
      // Try to extract from SKILL.md
      const skillPath = path.join(dirPath, 'SKILL.md');
      if (await this._fileExists(skillPath)) {
        const frontmatter = await this._extractYAMLFrontmatter(skillPath);
        if (frontmatter) {
          metadata.claude = frontmatter;
        }
      }

      // Try to extract from marketplace.json
      const marketplacePath = path.join(dirPath, '.claude-plugin', 'marketplace.json');
      if (await this._fileExists(marketplacePath)) {
        const content = await fs.readFile(marketplacePath, 'utf8');
        try {
          const json = JSON.parse(content);
          metadata.claudeMarketplace = json;
        } catch {}
      }
    }

    if (platform === PLATFORM_TYPES.GEMINI || platform === PLATFORM_TYPES.UNIVERSAL) {
      // Try to extract from gemini-extension.json
      const manifestPath = path.join(dirPath, 'gemini-extension.json');
      if (await this._fileExists(manifestPath)) {
        const content = await fs.readFile(manifestPath, 'utf8');
        try {
          const json = JSON.parse(content);
          metadata.gemini = json;
        } catch {}
      }
    }

    return metadata;
  }

  /**
   * Check if file exists
   */
  async _fileExists(filePath) {
    try {
      await fs.access(filePath);
      return true;
    } catch {
      return false;
    }
  }

  /**
   * Check if file is valid JSON
   */
  async _isValidJSON(filePath) {
    try {
      const content = await fs.readFile(filePath, 'utf8');
      JSON.parse(content);
      return true;
    } catch {
      return false;
    }
  }

  /**
   * Check if file has YAML frontmatter
   */
  async _hasYAMLFrontmatter(filePath) {
    try {
      const content = await fs.readFile(filePath, 'utf8');
      return /^---\n[\s\S]+?\n---/.test(content);
    } catch {
      return false;
    }
  }

  /**
   * Extract YAML frontmatter from file
   */
  async _extractYAMLFrontmatter(filePath) {
    try {
      const content = await fs.readFile(filePath, 'utf8');
      const match = content.match(/^---\n([\s\S]+?)\n---/);
      if (match) {
        // Simple YAML parser for basic key-value pairs
        const yaml = match[1];
        const parsed = {};

        const lines = yaml.split('\n');
        let currentKey = null;
        let currentValue = null;

        for (const line of lines) {
          if (line.trim().startsWith('-')) {
            // Array item
            if (currentKey && Array.isArray(parsed[currentKey])) {
              parsed[currentKey].push(line.trim().substring(1).trim());
            }
          } else if (line.includes(':')) {
            // Key-value pair
            const [key, ...valueParts] = line.split(':');
            const value = valueParts.join(':').trim();
            currentKey = key.trim();

            if (value === '') {
              // Array or multi-line value
              parsed[currentKey] = [];
            } else {
              parsed[currentKey] = value;
            }
          }
        }

        return parsed;
      }
      return null;
    } catch {
      return null;
    }
  }
}

export default PlatformDetector;
```

src/analyzers/validator.js
```
/**
 * Validation Utilities
 * Validates that converted skills/extensions meet platform requirements
 */

import fs from 'fs/promises';
import path from 'path';
import { PLATFORM_TYPES } from './detector.js';

export class Validator {
  constructor() {
    this.errors = [];
    this.warnings = [];
  }

  /**
   * Validate a skill/extension for a specific platform
   * @param {string} dirPath - Path to the directory to validate
   * @param {string} platform - Target platform (claude, gemini, or universal)
   * @returns {Promise<{valid: boolean, errors: array, warnings: array}>}
   */
  async validate(dirPath, platform) {
    this.errors = [];
    this.warnings = [];

    try {
      if (platform === PLATFORM_TYPES.CLAUDE || platform === PLATFORM_TYPES.UNIVERSAL) {
        await this._validateClaude(dirPath);
      }

      if (platform === PLATFORM_TYPES.GEMINI || platform === PLATFORM_TYPES.UNIVERSAL) {
        await this._validateGemini(dirPath);
      }

      return {
        valid: this.errors.length === 0,
        errors: this.errors,
        warnings: this.warnings
      };
    } catch (error) {
      this.errors.push(`Validation failed: ${error.message}`);
      return {
        valid: false,
        errors: this.errors,
        warnings: this.warnings
      };
    }
  }

  /**
   * Validate Claude skill requirements
   */
  async _validateClaude(dirPath) {
    // Check for SKILL.md
    const skillPath = path.join(dirPath, 'SKILL.md');
    if (!await this._fileExists(skillPath)) {
      this.errors.push('Missing required file: SKILL.md');
      return;
    }

    // Validate SKILL.md frontmatter
    const content = await fs.readFile(skillPath, 'utf8');
    const frontmatterMatch = content.match(/^---\n([\s\S]+?)\n---/);

    if (!frontmatterMatch) {
      this.errors.push('SKILL.md must have YAML frontmatter');
      return;
    }

    const frontmatter = this._parseYAML(frontmatterMatch[1]);

    // Check required frontmatter fields
    if (!frontmatter.name) {
      this.errors.push('SKILL.md frontmatter missing required field: name');
    } else {
      // Validate name format
      if (!/^[a-z0-9-]+$/.test(frontmatter.name)) {
        this.errors.push('Skill name must be lowercase letters, numbers, and hyphens only');
      }
      if (frontmatter.name.length > 64) {
        this.errors.push('Skill name must be 64 characters or less');
      }
    }

    if (!frontmatter.description) {
      this.errors.push('SKILL.md frontmatter missing required field: description');
    } else {
      if (frontmatter.description.length > 1024) {
        this.errors.push('Description must be 1024 characters or less');
      }
      if (frontmatter.description.length < 50) {
        this.warnings.push('Description should be descriptive (at least 50 characters recommended)');
      }
    }

    // Check for marketplace.json (optional but recommended)
    const marketplacePath = path.join(dirPath, '.claude-plugin', 'marketplace.json');
    if (!await this._fileExists(marketplacePath)) {
      this.warnings.push('Missing .claude-plugin/marketplace.json (recommended for MCP server integration)');
    } else {
      await this._validateMarketplaceJSON(marketplacePath);
    }

    // Validate file paths use forward slashes
    if (content.includes('\\')) {
      this.warnings.push('Use forward slashes (/) for file paths, not backslashes (\\)');
    }
  }

  /**
   * Validate Gemini extension requirements
   */
  async _validateGemini(dirPath) {
    // Check for gemini-extension.json
    const manifestPath = path.join(dirPath, 'gemini-extension.json');
    if (!await this._fileExists(manifestPath)) {
      this.errors.push('Missing required file: gemini-extension.json');
      return;
    }

    // Validate manifest JSON
    const content = await fs.readFile(manifestPath, 'utf8');
    let manifest;

    try {
      manifest = JSON.parse(content);
    } catch (error) {
      this.errors.push(`Invalid JSON in gemini-extension.json: ${error.message}`);
      return;
    }

    // Check required fields
    if (!manifest.name) {
      this.errors.push('gemini-extension.json missing required field: name');
    } else {
      // Validate name matches directory
      const dirName = path.basename(dirPath);
      if (manifest.name !== dirName) {
        this.warnings.push(`Extension name "${manifest.name}" should match directory name "${dirName}"`);
      }
    }

    if (!manifest.version) {
      this.errors.push('gemini-extension.json missing required field: version');
    }

    // Validate MCP servers configuration
    if (manifest.mcpServers) {
      for (const [serverName, config] of Object.entries(manifest.mcpServers)) {
        if (!config.command) {
          this.errors.push(`MCP server "${serverName}" missing required field: command`);
        }

        if (config.args) {
          // Check for proper variable substitution
          const argsStr = JSON.stringify(config.args);
          if (argsStr.includes('mcp-server') && !argsStr.includes('${extensionPath}')) {
            this.warnings.push(`MCP server "${serverName}" should use \${extensionPath} variable for paths`);
          }
        }
      }
    }

    // Validate settings if present
    if (manifest.settings) {
      if (!Array.isArray(manifest.settings)) {
        this.errors.push('settings must be an array');
      } else {
        manifest.settings.forEach((setting, index) => {
          if (!setting.name) {
            this.errors.push(`Setting at index ${index} missing required field: name`);
          }
          if (!setting.description) {
            this.warnings.push(`Setting "${setting.name}" should have a description`);
          }
        });
      }
    }

    // Check for context file
    const contextFileName = manifest.contextFileName || 'GEMINI.md';
    const contextPath = path.join(dirPath, contextFileName);
    if (!await this._fileExists(contextPath)) {
      this.warnings.push(`Missing context file: ${contextFileName} (recommended for providing context to Gemini)`);
    }

    // Validate excludeTools if present
    if (manifest.excludeTools) {
      if (!Array.isArray(manifest.excludeTools)) {
        this.errors.push('excludeTools must be an array');
      }
    }
  }

  /**
   * Validate marketplace.json structure
   */
  async _validateMarketplaceJSON(filePath) {
    const content = await fs.readFile(filePath, 'utf8');
    let marketplace;

    try {
      marketplace = JSON.parse(content);
    } catch (error) {
      this.errors.push(`Invalid JSON in marketplace.json: ${error.message}`);
      return;
    }

    // Check required fields
    if (!marketplace.name) {
      this.errors.push('marketplace.json missing required field: name');
    }

    if (!marketplace.metadata) {
      this.errors.push('marketplace.json missing required field: metadata');
    } else {
      if (!marketplace.metadata.description) {
        this.warnings.push('marketplace.json metadata should include description');
      }
      if (!marketplace.metadata.version) {
        this.warnings.push('marketplace.json metadata should include version');
      }
    }

    if (!marketplace.plugins || !Array.isArray(marketplace.plugins)) {
      this.errors.push('marketplace.json missing required field: plugins (array)');
    } else {
      marketplace.plugins.forEach((plugin, index) => {
        if (!plugin.name) {
          this.errors.push(`Plugin at index ${index} missing required field: name`);
        }
        if (!plugin.description) {
          this.errors.push(`Plugin at index ${index} missing required field: description`);
        }
      });
    }
  }

  /**
   * Simple YAML parser for validation
   */
  _parseYAML(yaml) {
    const parsed = {};
    const lines = yaml.split('\n');
    let currentKey = null;

    for (const line of lines) {
      if (line.trim().startsWith('-')) {
        // Array item
        if (currentKey && Array.isArray(parsed[currentKey])) {
          parsed[currentKey].push(line.trim().substring(1).trim());
        }
      } else if (line.includes(':')) {
        // Key-value pair
        const [key, ...valueParts] = line.split(':');
        const value = valueParts.join(':').trim();
        currentKey = key.trim();

        if (value === '') {
          // Array or multi-line value
          parsed[currentKey] = [];
        } else {
          parsed[currentKey] = value;
        }
      }
    }

    return parsed;
  }

  /**
   * Check if file exists
   */
  async _fileExists(filePath) {
    try {
      await fs.access(filePath);
      return true;
    } catch {
      return false;
    }
  }
}

export default Validator;
```

src/converters/claude-to-gemini.js
```
/**
 * Claude to Gemini Converter
 * Converts Claude Code skills to Gemini CLI extensions
 */

import fs from 'fs/promises';
import path from 'path';
import yaml from 'js-yaml';

export class ClaudeToGeminiConverter {
  constructor(sourcePath, outputPath) {
    this.sourcePath = sourcePath;
    this.outputPath = outputPath || sourcePath;
    this.metadata = {
      source: {},
      generated: []
    };
  }

  /**
   * Perform the conversion
   * @returns {Promise<{success: boolean, files: array, warnings: array}>}
   */
  async convert() {
    const result = {
      success: false,
      files: [],
      warnings: [],
      errors: []
    };

    try {
      // Ensure output directory exists
      await fs.mkdir(this.outputPath, { recursive: true });

      // Step 1: Extract metadata from Claude skill
      await this._extractClaudeMetadata();

      // Step 2: Generate gemini-extension.json
      const manifestPath = await this._generateGeminiManifest();
      result.files.push(manifestPath);

      // Step 3: Generate GEMINI.md from SKILL.md
      const contextPath = await this._generateGeminiContext();
      result.files.push(contextPath);

      // Step 4: Generate Custom Commands (from Subagents & Slash Commands)
      const commandFiles = await this._generateCommands();
      result.files.push(...commandFiles);

      // Step 5: Transform MCP server configuration
      await this._transformMCPConfiguration();

      // Step 6: Create shared directory structure
      await this._ensureSharedStructure();

      // Step 7: Inject Documentation
      await this._injectDocs();

      result.success = true;
      result.metadata = this.metadata;
    } catch (error) {
      result.success = false;
      result.errors.push(error.message);
    }

    return result;
  }

  /**
   * Extract metadata from Claude skill files
   */
  async _extractClaudeMetadata() {
    // Extract from SKILL.md
    const skillPath = path.join(this.sourcePath, 'SKILL.md');
    const content = await fs.readFile(skillPath, 'utf8');

    // Extract YAML frontmatter
    const frontmatterMatch = content.match(/^---\n([\s\S]+?)\n---/);
    if (!frontmatterMatch) {
      throw new Error('SKILL.md missing YAML frontmatter');
    }

    const frontmatter = yaml.load(frontmatterMatch[1]);
    this.metadata.source.frontmatter = frontmatter;

    // Extract content (without frontmatter)
    const contentWithoutFrontmatter = content.replace(/^---\n[\s\S]+?\n---\n/, '');
    this.metadata.source.content = contentWithoutFrontmatter;

    // Extract subagents if present
    if (frontmatter.subagents) {
      this.metadata.source.subagents = frontmatter.subagents;
    }

    // Extract Claude slash commands if present
    this.metadata.source.commands = [];
    const commandsDir = path.join(this.sourcePath, '.claude', 'commands');
    try {
      const files = await fs.readdir(commandsDir);
      for (const file of files) {
        if (file.endsWith('.md')) {
          const cmdPath = path.join(commandsDir, file);
          const cmdContent = await fs.readFile(cmdPath, 'utf8');
          this.metadata.source.commands.push({
            name: path.basename(file, '.md'),
            content: cmdContent
          });
        }
      }
    } catch {
      // No commands directory
    }

    // Extract from marketplace.json if it exists
    const marketplacePath = path.join(this.sourcePath, '.claude-plugin', 'marketplace.json');
    try {
      const marketplaceContent = await fs.readFile(marketplacePath, 'utf8');
      this.metadata.source.marketplace = JSON.parse(marketplaceContent);
    } catch {
      // marketplace.json is optional
      this.metadata.source.marketplace = null;
    }
  }

  /**
   * Generate gemini-extension.json
   */
  async _generateGeminiManifest() {
    const frontmatter = this.metadata.source.frontmatter;
    const marketplace = this.metadata.source.marketplace;

    // Build the manifest
    const manifest = {
      name: frontmatter.name,
      version: marketplace?.metadata?.version || '1.0.0',
      description: frontmatter.description || marketplace?.plugins?.[0]?.description || '',
      contextFileName: 'GEMINI.md'
    };

    // Transform MCP servers configuration
    if (marketplace?.plugins?.[0]?.mcpServers) {
      manifest.mcpServers = this._transformMCPServers(marketplace.plugins[0].mcpServers);
    }

    // Convert allowed-tools to excludeTools
    if (frontmatter['allowed-tools']) {
      manifest.excludeTools = this._convertAllowedToolsToExclude(frontmatter['allowed-tools']);
    }

    // Generate settings from MCP server environment variables
    if (manifest.mcpServers) {
      const settings = this._inferSettingsFromMCPConfig(manifest.mcpServers);
      if (settings.length > 0) {
        manifest.settings = settings;
      }
    }

    // Write to file
    const outputPath = path.join(this.outputPath, 'gemini-extension.json');
    await fs.writeFile(outputPath, JSON.stringify(manifest, null, 2));

    return outputPath;
  }

  /**
   * Transform MCP servers configuration for Gemini
   */
  _transformMCPServers(mcpServers) {
    const transformed = {};

    for (const [serverName, config] of Object.entries(mcpServers)) {
      transformed[serverName] = {
        ...config
      };

      // Transform args to use ${extensionPath}
      if (config.args) {
        transformed[serverName].args = config.args.map(arg => {
          // If it's a relative path, prepend ${extensionPath}
          if (arg.match(/^[a-z]/i) && !arg.startsWith('${')) {
            return `\${extensionPath}/${arg}`;
          }
          return arg;
        });
      }

      // Transform env variables to use settings
      if (config.env) {
        const newEnv = {};
        for (const [key, value] of Object.entries(config.env)) {
          // If it references an env var (${VAR}), keep it as is for settings
          if (typeof value === 'string' && value.match(/\$\{.+\}/)) {
            const varName = value.match(/\$\{(.+)\}/)[1];
            newEnv[key] = `\${${varName}}`;
          } else {
            newEnv[key] = value;
          }
        }
        transformed[serverName].env = newEnv;
      }
    }

    return transformed;
  }

  /**
   * Convert Claude's allowed-tools (whitelist) to Gemini's excludeTools (blacklist)
   */
  _convertAllowedToolsToExclude(allowedTools) {
    // List of all available tools
    const allTools = [
      'Read', 'Write', 'Edit', 'Glob', 'Grep', 'Bash', 'Task',
      'WebFetch', 'WebSearch', 'TodoWrite', 'AskUserQuestion',
      'SlashCommand', 'Skill', 'NotebookEdit', 'BashOutput', 'KillShell'
    ];

    // Normalize allowed tools to array
    let allowed = [];
    if (Array.isArray(allowedTools)) {
      allowed = allowedTools;
    } else if (typeof allowedTools === 'string') {
      allowed = allowedTools.split(',').map(t => t.trim());
    }

    // Calculate excluded tools
    const excluded = allTools.filter(tool => !allowed.includes(tool));

    // Generate exclude patterns
    // For Gemini, we can use simpler exclusions or keep it empty if minimal restrictions
    // Return empty array if most tools are allowed (simpler approach)
    if (excluded.length > allowed.length) {
      // If more tools are excluded than allowed, return exclude list
      return excluded;
    } else {
      // If more tools are allowed, we can't easily express this in Gemini
      // Return empty and add a warning
      this.metadata.warnings = this.metadata.warnings || [];
      this.metadata.warnings.push('Tool restrictions may not translate exactly - review excludeTools in gemini-extension.json');
      return [];
    }
  }

  /**
   * Infer settings schema from MCP server environment variables
   */
  _inferSettingsFromMCPConfig(mcpServers) {
    const settings = [];
    const seenVars = new Set();

    for (const [, config] of Object.entries(mcpServers)) {
      if (config.env) {
        for (const [key, value] of Object.entries(config.env)) {
          // Extract variable name from ${VAR} pattern
          if (typeof value === 'string' && value.match(/\$\{(.+)\}/)) {
            const varName = value.match(/\$\{(.+)\}/)[1];

            // Skip if already seen
            if (seenVars.has(varName)) continue;
            seenVars.add(varName);

            // Infer setting properties
            const setting = {
              name: varName,
              description: this._inferDescription(varName)
            };

            // Detect if it's a secret/password
            if (varName.toLowerCase().includes('password') ||
                varName.toLowerCase().includes('secret') ||
                varName.toLowerCase().includes('token') ||
                varName.toLowerCase().includes('key')) {
              setting.secret = true;
              setting.required = true;
            }

            // Add default values for common settings
            const defaults = this._inferDefaults(varName);
            if (defaults) {
              Object.assign(setting, defaults);
            }

            settings.push(setting);
          }
        }
      }
    }

    return settings;
  }

  /**
   * Infer description from variable name
   */
  _inferDescription(varName) {
    const descriptions = {
      'DB_HOST': 'Database server hostname',
      'DB_PORT': 'Database server port',
      'DB_NAME': 'Database name',
      'DB_USER': 'Database username',
      'DB_PASSWORD': 'Database password',
      'API_KEY': 'API authentication key',
      'API_SECRET': 'API secret',
      'API_URL': 'API endpoint URL',
      'HOST': 'Server hostname',
      'PORT': 'Server port'
    };

    if (descriptions[varName]) {
      return descriptions[varName];
    }

    // Generate description from variable name
    return varName.split('_')
      .map(word => word.charAt(0) + word.slice(1).toLowerCase())
      .join(' ');
  }

  /**
   * Infer default values for common variables
   */
  _inferDefaults(varName) {
    const defaults = {
      'DB_HOST': { default: 'localhost' },
      'DB_PORT': { default: '5432' },
      'HOST': { default: 'localhost' },
      'PORT': { default: '8080' },
      'API_URL': { default: 'https://api.example.com' }
    };

    return defaults[varName] || null;
  }

  /**
   * Generate GEMINI.md from SKILL.md content
   */
  async _generateGeminiContext() {
    const content = this.metadata.source.content;
    const frontmatter = this.metadata.source.frontmatter;

    // Build Gemini context with platform-specific introduction
    let geminiContent = `# ${frontmatter.name} - Gemini CLI Extension\n\n`;
    geminiContent += `${frontmatter.description}\n\n`;
    geminiContent += `## Quick Start\n\nAfter installation, you can use this extension by asking questions or giving commands naturally.\n\n`;

    // Add original content
    geminiContent += content;

    // Add footer
    geminiContent += `\n\n---\n\n`;
    geminiContent += `*This extension was converted from a Claude Code skill using [skill-porter](https://github.com/jduncan-rva/skill-porter)*\n`;

    // Write to file
    const outputPath = path.join(this.outputPath, 'GEMINI.md');
    await fs.writeFile(outputPath, geminiContent);

    return outputPath;
  }

  /**
   * Generate Gemini Custom Commands
   */
  async _generateCommands() {
    const generatedFiles = [];
    const commandsDir = path.join(this.outputPath, 'commands');
    
    // Ensure commands directory exists if we have content
    const subagents = this.metadata.source.subagents || [];
    const commands = this.metadata.source.commands || [];
    
    if (subagents.length === 0 && commands.length === 0) {
      return generatedFiles;
    }
    
    await fs.mkdir(commandsDir, { recursive: true });

    // Convert Subagents -> Commands
    for (const agent of subagents) {
      const tomlContent = `description = "Activate ${agent.name} agent"

# Agent Persona: ${agent.name}
# Auto-generated from Claude Subagent
prompt = """
You are acting as the '${agent.name}' agent.
${agent.description || ''}

User Query: {{args}}
"""
`;
      const filePath = path.join(commandsDir, `${agent.name}.toml`);
      await fs.writeFile(filePath, tomlContent);
      generatedFiles.push(filePath);
    }

    // Convert Claude Commands -> Gemini Commands
    for (const cmd of commands) {
      // Extract frontmatter from command if present
      const match = cmd.content.match(/^---\n([\s\S]+?)\n---\n([\s\S]+)$/);
      let description = `Custom command: ${cmd.name}`;
      let prompt = cmd.content;

      if (match) {
        try {
          const fm = yaml.load(match[1]);
          if (fm.description) description = fm.description;
          prompt = match[2]; // Content without frontmatter
        } catch (e) {
          // Fallback if YAML invalid
        }
      }

      // Convert arguments syntax
      // Claude: $ARGUMENTS, $1, etc. -> Gemini: {{args}}
      prompt = prompt.replace(/\$ARGUMENTS/g, '{{args}}')
                     .replace(/\$\d+/g, '{{args}}');

      const tomlContent = `description = "${description}"

prompt = """
${prompt.trim()}
"""
`;
      const filePath = path.join(commandsDir, `${cmd.name}.toml`);
      await fs.writeFile(filePath, tomlContent);
      generatedFiles.push(filePath);
    }

    return generatedFiles;
  }

  /**
   * Inject Architecture Documentation
   */
  async _injectDocs() {
    const docsDir = path.join(this.outputPath, 'docs');
    await fs.mkdir(docsDir, { recursive: true });

    // Path to the template we created earlier
    // Assuming the CLI is run from the root where templates/ exists
    // In a real package, this should be resolved relative to __dirname
    const templatePath = path.resolve('templates', 'GEMINI_ARCH_GUIDE.md');
    const destPath = path.join(docsDir, 'GEMINI_ARCHITECTURE.md');

    try {
      const content = await fs.readFile(templatePath, 'utf8');
      await fs.writeFile(destPath, content);
    } catch (error) {
      // Fallback if template missing (e.g. in dev environment vs prod)
      await fs.writeFile(destPath, '# Gemini Architecture\n\nSee online documentation.');
    }
  }

  /**
   * Transform MCP configuration files
   */
  async _transformMCPConfiguration() {
    // Check if mcp-server directory exists
    const mcpDir = path.join(this.sourcePath, 'mcp-server');
    try {
      await fs.access(mcpDir);
      // MCP server exists and is already shared - no changes needed
    } catch {
      // No MCP server directory - this is okay
    }
  }

  /**
   * Ensure shared directory structure exists
   */
  async _ensureSharedStructure() {
    const sharedDir = path.join(this.outputPath, 'shared');

    try {
      await fs.access(sharedDir);
      // Directory exists
    } catch {
      // Create shared directory
      await fs.mkdir(sharedDir, { recursive: true });

      // Create placeholder files
      const referenceContent = `# Technical Reference

## Architecture
For detailed extension architecture, please refer to \`docs/GEMINI_ARCHITECTURE.md\` (in Gemini extensions) or the \`SKILL.md\` structure (in Claude Skills).

## Platform Differences
- **Commands:**
  - Gemini uses \`commands/*.toml\`
  - Claude uses \`.claude/commands/*.md\`
- **Agents:**
  - Gemini "Agents" are implemented as Custom Commands.
  - Claude "Subagents" are defined in \`SKILL.md\` frontmatter.
`;
      await fs.writeFile(
        path.join(sharedDir, 'reference.md'),
        referenceContent
      );

      await fs.writeFile(
        path.join(sharedDir, 'examples.md'),
        '# Usage Examples\n\nComprehensive usage examples and tutorials.\n'
      );
    }
  }
}

export default ClaudeToGeminiConverter;
```

src/converters/gemini-to-claude.js
```
/**
 * Gemini to Claude Converter
 * Converts Gemini CLI extensions to Claude Code skills
 */

import fs from 'fs/promises';
import path from 'path';
import yaml from 'js-yaml';

export class GeminiToClaudeConverter {
  constructor(sourcePath, outputPath) {
    this.sourcePath = sourcePath;
    this.outputPath = outputPath || sourcePath;
    this.metadata = {
      source: {},
      generated: []
    };
  }

  /**
   * Perform the conversion
   * @returns {Promise<{success: boolean, files: array, warnings: array}>}
   */
  async convert() {
    const result = {
      success: false,
      files: [],
      warnings: [],
      errors: []
    };

    try {
      // Ensure output directory exists
      await fs.mkdir(this.outputPath, { recursive: true });

      // Step 1: Extract metadata from Gemini extension
      await this._extractGeminiMetadata();

      // Step 2: Generate SKILL.md
      const skillPath = await this._generateClaudeSkill();
      result.files.push(skillPath);

      // Step 3: Generate .claude-plugin/marketplace.json
      const marketplacePath = await this._generateMarketplaceJSON();
      result.files.push(marketplacePath);

      // Step 4: Generate Custom Commands
      const commandFiles = await this._generateClaudeCommands();
      result.files.push(...commandFiles);

      // Step 5: Transform MCP server configuration
      await this._transformMCPConfiguration();

      // Step 6: Create shared directory structure if it doesn't exist
      await this._ensureSharedStructure();

      // Step 7: Generate Insights
      await this._generateMigrationInsights();

      result.success = true;
      result.metadata = this.metadata;
    } catch (error) {
      result.success = false;
      result.errors.push(error.message);
    }

    return result;
  }

  /**
   * Extract metadata from Gemini extension files
   */
  async _extractGeminiMetadata() {
    // Extract from gemini-extension.json
    const manifestPath = path.join(this.sourcePath, 'gemini-extension.json');
    const manifestContent = await fs.readFile(manifestPath, 'utf8');
    this.metadata.source.manifest = JSON.parse(manifestContent);

    // Extract from GEMINI.md or custom context file
    const contextFileName = this.metadata.source.manifest.contextFileName || 'GEMINI.md';
    const contextPath = path.join(this.sourcePath, contextFileName);

    try {
      const content = await fs.readFile(contextPath, 'utf8');
      this.metadata.source.content = content;
    } catch {
      // Context file is optional
      this.metadata.source.content = '';
    }

    // Extract commands if present
    this.metadata.source.commands = [];
    const commandsDir = path.join(this.sourcePath, 'commands');
    try {
      const files = await fs.readdir(commandsDir);
      for (const file of files) {
        if (file.endsWith('.toml')) {
          const cmdPath = path.join(commandsDir, file);
          const cmdContent = await fs.readFile(cmdPath, 'utf8');
          this.metadata.source.commands.push({
            name: path.basename(file, '.toml'),
            content: cmdContent
          });
        }
      }
    } catch {
      // No commands directory
    }
  }

  /**
   * Generate SKILL.md with YAML frontmatter
   */
  async _generateClaudeSkill() {
    const manifest = this.metadata.source.manifest;
    const content = this.metadata.source.content;

    // Build frontmatter
    const frontmatter = {
      name: manifest.name,
      description: manifest.description
    };

    // Convert excludeTools to allowed-tools
    if (manifest.excludeTools && manifest.excludeTools.length > 0) {
      frontmatter['allowed-tools'] = this._convertExcludeToAllowedTools(manifest.excludeTools);
    }

    // Convert frontmatter to YAML
    const yamlFrontmatter = yaml.dump(frontmatter, {
      lineWidth: -1, // Disable line wrapping
      noArrayIndent: false
    });

    // Build SKILL.md content
    let skillContent = `---\n${yamlFrontmatter}---\n\n`;

    // Add title and description
    skillContent += `# ${manifest.name} - Claude Code Skill\n\n`;
    skillContent += `${manifest.description}\n\n`;

    // Add original content (without Gemini-specific header if present)
    let cleanContent = content;

    // Remove Gemini-specific headers
    cleanContent = cleanContent.replace(/^#\s+.+?\s+-\s+Gemini CLI Extension\n\n/m, '');
    cleanContent = cleanContent.replace(/##\s+Quick Start[\s\S]+?After installation.+?\n\n/m, '');

    // Remove conversion footer if present
    cleanContent = cleanContent.replace(/\n---\n\n\*This extension was converted.+?\*\n$/s, '');

    // Add environment variable configuration section if there are settings
    if (manifest.settings && manifest.settings.length > 0) {
      skillContent += `## Configuration\n\nThis skill requires the following environment variables:\n\n`;

      for (const setting of manifest.settings) {
        skillContent += `- \`${setting.name}\`: ${setting.description}`;
        if (setting.default) {
          skillContent += ` (default: ${setting.default})`;
        }
        if (setting.required) {
          skillContent += ` **(required)**`;
        }
        skillContent += `\n`;
      }

      skillContent += `\nSet these in your environment or Claude Code configuration.\n\n`;
    }

    // Add cleaned content
    if (cleanContent.trim()) {
      skillContent += cleanContent.trim() + '\n\n';
    } else {
      // Generate basic usage section if no content
      skillContent += `## Usage\n\nUse this skill when you need ${manifest.description.toLowerCase()}.\n\n`;
    }

    // Add footer
    skillContent += `---\n\n`;
    skillContent += `*This skill was converted from a Gemini CLI extension using [skill-porter](https://github.com/jduncan-rva/skill-porter)*\n`;

    // Write to file
    const outputPath = path.join(this.outputPath, 'SKILL.md');
    await fs.writeFile(outputPath, skillContent);

    return outputPath;
  }

  /**
   * Convert Gemini's excludeTools (blacklist) to Claude's allowed-tools (whitelist)
   */
  _convertExcludeToAllowedTools(excludeTools) {
    // List of all available tools
    const allTools = [
      'Read', 'Write', 'Edit', 'Glob', 'Grep', 'Bash', 'Task',
      'WebFetch', 'WebSearch', 'TodoWrite', 'AskUserQuestion',
      'SlashCommand', 'Skill', 'NotebookEdit', 'BashOutput', 'KillShell'
    ];

    // Calculate allowed tools (all tools minus excluded)
    const allowed = allTools.filter(tool => !excludeTools.includes(tool));

    return allowed;
  }

  /**
   * Generate .claude-plugin/marketplace.json
   */
  async _generateMarketplaceJSON() {
    const manifest = this.metadata.source.manifest;

    // Build marketplace.json
    const marketplace = {
      name: `${manifest.name}-marketplace`,
      owner: {
        name: 'Skill Porter User',
        email: 'user@example.com'
      },
      metadata: {
        description: manifest.description,
        version: manifest.version || '1.0.0'
      },
      plugins: [
        {
          name: manifest.name,
          description: manifest.description,
          source: '.',
          strict: false,
          author: 'Converted from Gemini',
          repository: {
            type: 'git',
            url: `https://github.com/user/${manifest.name}`
          },
          license: 'MIT',
          keywords: this._extractKeywords(manifest.description),
          category: 'general',
          tags: [],
          skills: ['.']
        }
      ]
    };

    // Add MCP servers configuration if present
    if (manifest.mcpServers) {
      marketplace.plugins[0].mcpServers = this._transformMCPServersForClaude(manifest.mcpServers, manifest.settings);
    }

    // Create .claude-plugin directory
    const claudePluginDir = path.join(this.outputPath, '.claude-plugin');
    await fs.mkdir(claudePluginDir, { recursive: true });

    // Write to file
    const outputPath = path.join(claudePluginDir, 'marketplace.json');
    await fs.writeFile(outputPath, JSON.stringify(marketplace, null, 2));

    return outputPath;
  }

  /**
   * Generate Claude Custom Commands
   */
  async _generateClaudeCommands() {
    const generatedFiles = [];
    const commands = this.metadata.source.commands || [];
    
    if (commands.length === 0) {
      return generatedFiles;
    }
    
    const commandsDir = path.join(this.outputPath, '.claude', 'commands');
    await fs.mkdir(commandsDir, { recursive: true });

    for (const cmd of commands) {
      // Simple TOML parsing (regex based to avoid dependency for now)
      const descMatch = cmd.content.match(/description\s*=\s*"([^"]+)"/);
      const promptMatch = cmd.content.match(/prompt\s*=\s*"""([\s\S]+?)"""/);

      const description = descMatch ? descMatch[1] : `Run ${cmd.name}`;
      let prompt = promptMatch ? promptMatch[1] : '';

      // Convert arguments syntax
      // Gemini: {{args}} -> Claude: $ARGUMENTS
      prompt = prompt.replace(/\{\{args\}\}/g, '$ARGUMENTS');

      const mdContent = `---
description: ${description}
---

${prompt.trim()}
`;
      const filePath = path.join(commandsDir, `${cmd.name}.md`);
      await fs.writeFile(filePath, mdContent);
      generatedFiles.push(filePath);
    }

    return generatedFiles;
  }

  /**
   * Generate Migration Insights Report
   */
  async _generateMigrationInsights() {
    const commands = this.metadata.source.commands || [];
    const insights = [];
    const sharedDir = path.join(this.outputPath, 'shared');

    // Heuristic checks
    for (const cmd of commands) {
      const prompt = (cmd.content.match(/prompt\s*=\s*"""([\s\S]+?)"""/) || [])[1] || '';
      
      // Check for Persona definition
      if (prompt.match(/You are a|Act as|Your role is/i)) {
        insights.push({
          type: 'PERSONA_DETECTED',
          command: cmd.name,
          message: `Command \`/${cmd.name}\` appears to define a persona. Consider moving this logic to \`SKILL.md\` instructions so Claude can adopt it automatically without a slash command.`
        });
      }
    }

    // Generate Report Content
    let content = `# Migration Insights & Recommendations\n\n`;
    content += `Generated during conversion from Gemini to Claude.\n\n`;

    if (insights.length > 0) {
      content += `## 💡 Optimization Opportunities\n\n`;
      content += `While we successfully converted your commands to Claude Slash Commands, some might work better as native Skill instructions.\n\n`;
      
      for (const insight of insights) {
        content += `### \`/${insight.command}\`\n`;
        content += `${insight.message}\n\n`;
      }
      
      content += `## How to Apply\n`;
      content += `1. Open \`SKILL.md\`\n`;
      content += `2. Paste the prompt instructions into the main description area.\n`;
      content += `3. Delete \`.claude/commands/${insights[0].command}.md\` if you prefer automatic invocation.\n`;
    } else {
      content += `✅ No specific architectural changes recommended. The direct conversion should work well.\n`;
    }

    await fs.writeFile(path.join(sharedDir, 'MIGRATION_INSIGHTS.md'), content);
  }

  /**
   * Transform MCP servers configuration for Claude
   */
  _transformMCPServersForClaude(mcpServers, settings) {
    const transformed = {};

    for (const [serverName, config] of Object.entries(mcpServers)) {
      transformed[serverName] = {
        ...config
      };

      // Transform args to remove ${extensionPath}
      if (config.args) {
        transformed[serverName].args = config.args.map(arg => {
          // Remove ${extensionPath}/ prefix
          return arg.replace(/\$\{extensionPath\}\//g, '');
        });
      }

      // Transform env to use ${VAR} pattern
      if (config.env) {
        const newEnv = {};
        for (const [key, value] of Object.entries(config.env)) {
          // If it uses a settings variable, convert to ${VAR}
          if (typeof value === 'string' && value.match(/\$\{.+\}/)) {
            newEnv[key] = value; // Keep as is
          } else {
            newEnv[key] = value;
          }
        }
        transformed[serverName].env = newEnv;
      }
    }

    return transformed;
  }

  /**
   * Extract keywords from description
   */
  _extractKeywords(description) {
    // Simple keyword extraction
    const commonWords = ['the', 'a', 'an', 'and', 'or', 'but', 'for', 'with', 'to', 'from', 'in', 'on'];
    const words = description.toLowerCase()
      .split(/\s+/)
      .filter(word => word.length > 3 && !commonWords.includes(word))
      .slice(0, 5);

    return words;
  }

  /**
   * Transform MCP configuration files
   */
  async _transformMCPConfiguration() {
    // Check if mcp-server directory exists
    const mcpDir = path.join(this.sourcePath, 'mcp-server');
    try {
      await fs.access(mcpDir);
      // MCP server exists and is already shared - no changes needed
    } catch {
      // No MCP server directory - this is okay
    }
  }

  /**
   * Ensure shared directory structure exists
   */
  async _ensureSharedStructure() {
    const sharedDir = path.join(this.outputPath, 'shared');

    try {
      await fs.access(sharedDir);
      // Directory exists
    } catch {
      // Create shared directory
      await fs.mkdir(sharedDir, { recursive: true });

      // Create placeholder files
      const referenceContent = `# Technical Reference

## Architecture
For detailed extension architecture, please refer to \`docs/GEMINI_ARCHITECTURE.md\` (in Gemini extensions) or the \`SKILL.md\` structure (in Claude Skills).

## Platform Differences
- **Commands:**
  - Gemini uses \`commands/*.toml\`
  - Claude uses \`.claude/commands/*.md\`
- **Agents:**
  - Gemini "Agents" are implemented as Custom Commands.
  - Claude "Subagents" are defined in \`SKILL.md\` frontmatter.
`;
      await fs.writeFile(
        path.join(sharedDir, 'reference.md'),
        referenceContent
      );

      await fs.writeFile(
        path.join(sharedDir, 'examples.md'),
        '# Usage Examples\n\nComprehensive usage examples and tutorials.\n'
      );
    }
  }
}

export default GeminiToClaudeConverter;
```

src/optional-features/fork-setup.js
```
/**
 * Fork Setup Feature
 * Creates a fork with dual-platform configuration for simultaneous use
 */

import { execSync } from 'child_process';
import fs from 'fs/promises';
import path from 'path';

export class ForkSetup {
  constructor(sourcePath) {
    this.sourcePath = sourcePath;
  }

  /**
   * Create a fork and set up for dual-platform use
   * @param {object} options - Fork setup options
   * @returns {Promise<{success: boolean, forkPath: string, errors: array}>}
   */
  async setup(options = {}) {
    const {
      forkLocation,
      repoUrl,
      branchName = 'dual-platform-setup'
    } = options;

    const result = {
      success: false,
      forkPath: null,
      errors: [],
      installations: {
        claude: null,
        gemini: null
      }
    };

    try {
      // Step 1: Validate inputs
      if (!forkLocation) {
        throw new Error('Fork location is required (use --fork-location)');
      }

      // Step 2: Create fork directory
      const forkPath = await this._createForkDirectory(forkLocation);
      result.forkPath = forkPath;

      // Step 3: Clone or copy repository
      if (repoUrl) {
        await this._cloneRepository(repoUrl, forkPath);
      } else {
        await this._copyDirectory(this.sourcePath, forkPath);
      }

      // Step 4: Ensure both platform configurations exist
      await this._ensureDualPlatform(forkPath);

      // Step 5: Set up installations
      const installations = await this._setupInstallations(forkPath);
      result.installations = installations;

      result.success = true;
    } catch (error) {
      result.errors.push(error.message);
    }

    return result;
  }

  /**
   * Create fork directory
   */
  async _createForkDirectory(forkLocation) {
    try {
      const resolvedPath = path.resolve(forkLocation);
      await fs.mkdir(resolvedPath, { recursive: true });
      return resolvedPath;
    } catch (error) {
      throw new Error(`Failed to create fork directory: ${error.message}`);
    }
  }

  /**
   * Clone repository from URL
   */
  async _cloneRepository(repoUrl, forkPath) {
    try {
      execSync(`git clone ${repoUrl} ${forkPath}`, {
        stdio: 'inherit'
      });
    } catch (error) {
      throw new Error(`Failed to clone repository: ${error.message}`);
    }
  }

  /**
   * Copy directory recursively
   */
  async _copyDirectory(source, destination) {
    try {
      // Use cp command for efficient copying
      execSync(`cp -r "${source}" "${destination}"`, {
        stdio: 'inherit'
      });
    } catch (error) {
      throw new Error(`Failed to copy directory: ${error.message}`);
    }
  }

  /**
   * Ensure both platform configurations exist
   */
  async _ensureDualPlatform(forkPath) {
    const hasClaudeConfig = await this._checkFileExists(path.join(forkPath, 'SKILL.md'));
    const hasGeminiConfig = await this._checkFileExists(path.join(forkPath, 'gemini-extension.json'));

    if (hasClaudeConfig && hasGeminiConfig) {
      // Already universal
      return;
    }

    // Need to convert
    const SkillPorter = (await import('../index.js')).default;
    const porter = new SkillPorter();

    if (!hasGeminiConfig) {
      // Convert to Gemini
      await porter.convert(forkPath, 'gemini', { validate: true });
    }

    if (!hasClaudeConfig) {
      // Convert to Claude
      await porter.convert(forkPath, 'claude', { validate: true });
    }
  }

  /**
   * Set up installations for both platforms
   */
  async _setupInstallations(forkPath) {
    const installations = {
      claude: null,
      gemini: null
    };

    // Get skill/extension name
    const skillName = path.basename(forkPath);

    // Set up Claude installation (symlink to personal skills directory)
    const claudeSkillPath = path.join(process.env.HOME, '.claude', 'skills', skillName);
    try {
      // Check if Claude skills directory exists
      await fs.mkdir(path.join(process.env.HOME, '.claude', 'skills'), { recursive: true });

      // Create symlink
      try {
        await fs.symlink(forkPath, claudeSkillPath, 'dir');
        installations.claude = claudeSkillPath;
      } catch (error) {
        if (error.code === 'EEXIST') {
          // Symlink already exists
          installations.claude = `${claudeSkillPath} (already exists)`;
        } else {
          throw error;
        }
      }
    } catch (error) {
      installations.claude = `Failed: ${error.message}`;
    }

    // For Gemini, we can't auto-install, but provide instructions
    installations.gemini = 'Run: gemini extensions install ' + forkPath;

    return installations;
  }

  /**
   * Check if file exists
   */
  async _checkFileExists(filePath) {
    try {
      await fs.access(filePath);
      return true;
    } catch {
      return false;
    }
  }
}

export default ForkSetup;
```

src/optional-features/pr-generator.js
```
/**
 * PR Generation Feature
 * Creates pull requests to add dual-platform support to repositories
 */

import { execSync } from 'child_process';
import fs from 'fs/promises';
import path from 'path';

export class PRGenerator {
  constructor(sourcePath) {
    this.sourcePath = sourcePath;
    this.branchName = `skill-porter/add-dual-platform-support`;
  }

  /**
   * Generate a pull request for dual-platform support
   * @param {object} options - PR generation options
   * @returns {Promise<{success: boolean, prUrl: string, errors: array}>}
   */
  async generate(options = {}) {
    const {
      targetPlatform,
      remote = 'origin',
      baseBranch = 'main',
      draft = false
    } = options;

    const result = {
      success: false,
      prUrl: null,
      errors: [],
      branch: this.branchName
    };

    try {
      // Step 1: Check if gh CLI is available
      await this._checkGHCLI();

      // Step 2: Check if we're in a git repository
      await this._checkGitRepo();

      // Step 3: Check for uncommitted changes
      const hasChanges = await this._hasUncommittedChanges();
      if (!hasChanges) {
        throw new Error('No uncommitted changes found. Run conversion first.');
      }

      // Step 4: Create new branch
      await this._createBranch();

      // Step 5: Commit changes
      await this._commitChanges(targetPlatform);

      // Step 6: Push branch
      await this._pushBranch(remote);

      // Step 7: Create PR
      const prUrl = await this._createPR(targetPlatform, baseBranch, draft);
      result.prUrl = prUrl;

      result.success = true;
    } catch (error) {
      result.errors.push(error.message);
    }

    return result;
  }

  /**
   * Check if gh CLI is installed
   */
  async _checkGHCLI() {
    try {
      execSync('gh --version', { stdio: 'ignore' });
    } catch {
      throw new Error('GitHub CLI (gh) not found. Install from https://cli.github.com');
    }

    // Check if authenticated
    try {
      execSync('gh auth status', { stdio: 'ignore' });
    } catch {
      throw new Error('GitHub CLI not authenticated. Run: gh auth login');
    }
  }

  /**
   * Check if directory is a git repository
   */
  async _checkGitRepo() {
    try {
      execSync('git rev-parse --git-dir', {
        cwd: this.sourcePath,
        stdio: 'ignore'
      });
    } catch {
      throw new Error('Not a git repository. Initialize with: git init');
    }
  }

  /**
   * Check for uncommitted changes
   */
  async _hasUncommittedChanges() {
    try {
      const status = execSync('git status --porcelain', {
        cwd: this.sourcePath,
        encoding: 'utf8'
      });
      return status.trim().length > 0;
    } catch {
      return false;
    }
  }

  /**
   * Create a new branch
   */
  async _createBranch() {
    try {
      // Check if branch already exists
      try {
        execSync(`git rev-parse --verify ${this.branchName}`, {
          cwd: this.sourcePath,
          stdio: 'ignore'
        });
        // Branch exists, check it out
        execSync(`git checkout ${this.branchName}`, {
          cwd: this.sourcePath,
          stdio: 'ignore'
        });
      } catch {
        // Branch doesn't exist, create it
        execSync(`git checkout -b ${this.branchName}`, {
          cwd: this.sourcePath,
          stdio: 'ignore'
        });
      }
    } catch (error) {
      throw new Error(`Failed to create branch: ${error.message}`);
    }
  }

  /**
   * Commit changes
   */
  async _commitChanges(targetPlatform) {
    const platformName = targetPlatform === 'gemini' ? 'Gemini CLI' : 'Claude Code';
    const otherPlatform = targetPlatform === 'gemini' ? 'Claude Code' : 'Gemini CLI';

    const commitMessage = `Add ${platformName} support for cross-platform compatibility

This PR adds ${platformName} support while maintaining existing ${otherPlatform} functionality, making this skill/extension work on both platforms.

## Changes

${targetPlatform === 'gemini' ? `
- Added \`gemini-extension.json\` - Gemini CLI manifest
- Added \`GEMINI.md\` - Gemini context file
- Created \`shared/\` directory for shared documentation
- Transformed MCP server paths for Gemini compatibility
- Converted tool restrictions (allowed-tools → excludeTools)
- Inferred settings schema from environment variables
` : `
- Added \`SKILL.md\` - Claude Code skill definition
- Added \`.claude-plugin/marketplace.json\` - Claude plugin config
- Created \`shared/\` directory for shared documentation
- Transformed MCP server paths for Claude compatibility
- Converted tool restrictions (excludeTools → allowed-tools)
- Documented environment variables from settings
`}

## Benefits

- ✅ Single codebase works on both AI platforms
- ✅ 85%+ code reuse (shared MCP server and docs)
- ✅ Easier maintenance (fix once, works everywhere)
- ✅ Broader user base (Claude + Gemini communities)

## Testing

- [x] Conversion validated with skill-porter
- [x] Files meet ${platformName} requirements
- [ ] Tested installation on ${platformName}
- [ ] Verified functionality on both platforms

## Installation

### ${otherPlatform} (existing)
\`\`\`bash
${otherPlatform === 'Claude Code' ?
  'cp -r . ~/.claude/skills/$(basename $PWD)' :
  'gemini extensions install .'}
\`\`\`

### ${platformName} (new)
\`\`\`bash
${platformName === 'Gemini CLI' ?
  'gemini extensions install .' :
  'cp -r . ~/.claude/skills/$(basename $PWD)'}
\`\`\`

---

*Generated with [skill-porter](https://github.com/jduncan-rva/skill-porter) - Universal tool for cross-platform AI skills*`;

    try {
      // Stage all new/modified files
      execSync('git add .', { cwd: this.sourcePath });

      // Create commit
      execSync(`git commit -m "${commitMessage.replace(/"/g, '\\"')}"`, {
        cwd: this.sourcePath,
        stdio: 'ignore'
      });
    } catch (error) {
      throw new Error(`Failed to commit changes: ${error.message}`);
    }
  }

  /**
   * Push branch to remote
   */
  async _pushBranch(remote) {
    try {
      execSync(`git push -u ${remote} ${this.branchName}`, {
        cwd: this.sourcePath,
        stdio: 'inherit'
      });
    } catch (error) {
      throw new Error(`Failed to push branch: ${error.message}`);
    }
  }

  /**
   * Create pull request
   */
  async _createPR(targetPlatform, baseBranch, draft) {
    const platformName = targetPlatform === 'gemini' ? 'Gemini CLI' : 'Claude Code';

    const title = `Add ${platformName} support for cross-platform compatibility`;
    const body = `This PR adds ${platformName} support, making this skill/extension work on both Claude Code and Gemini CLI.

## Overview

Converted using [skill-porter](https://github.com/jduncan-rva/skill-porter) to enable dual-platform deployment with minimal code duplication.

## What Changed

${targetPlatform === 'gemini' ? '✅ Added Gemini CLI support' : '✅ Added Claude Code support'}
- Platform-specific configuration files
- Shared documentation structure
- Converted tool restrictions and settings

## Benefits

- 🌐 Works on both AI platforms
- 🔄 85%+ code reuse
- 📦 Single repository
- 🚀 Easier maintenance

## Testing Checklist

- [x] Conversion validated
- [ ] Tested on ${platformName}
- [ ] Documentation updated

## Questions?

See the [skill-porter documentation](https://github.com/jduncan-rva/skill-porter) for details on universal skills.`;

    try {
      const draftFlag = draft ? '--draft' : '';
      const output = execSync(
        `gh pr create --base ${baseBranch} --title "${title}" --body "${body.replace(/"/g, '\\"')}" ${draftFlag}`,
        {
          cwd: this.sourcePath,
          encoding: 'utf8'
        }
      );

      // Extract PR URL from output
      const urlMatch = output.match(/https:\/\/github\.com\/[^\s]+/);
      if (urlMatch) {
        return urlMatch[0];
      }

      return 'PR created successfully';
    } catch (error) {
      throw new Error(`Failed to create PR: ${error.message}`);
    }
  }
}

export default PRGenerator;
```

</source_code>