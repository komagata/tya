import * as vscode from "vscode";
import { execFile, execFileSync } from "child_process";
import * as fs from "fs";
import * as os from "os";
import * as path from "path";
import { promisify } from "util";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;
const execFileAsync = promisify(execFile);

export function activate(context: vscode.ExtensionContext): void {
  const config = vscode.workspace.getConfiguration("tya");
  const exe = resolveExecutable(config.get<string>("executable", "tya"));
  context.subscriptions.push(registerFormatter(exe));
  const serverOptions: ServerOptions = {
    run: { command: exe, args: ["lsp"], transport: TransportKind.stdio },
    debug: { command: exe, args: ["lsp", "--log", "/tmp/tya-lsp.log"], transport: TransportKind.stdio },
  };
  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "tya" }],
    synchronize: {
      fileEvents: vscode.workspace.createFileSystemWatcher("**/*.tya"),
    },
  };
  client = new LanguageClient("tya", "tya", serverOptions, clientOptions);
  client.start();
}

export function deactivate(): Thenable<void> | undefined {
  return client?.stop();
}

function registerFormatter(exe: string): vscode.Disposable {
  return vscode.languages.registerDocumentFormattingEditProvider("tya", {
    async provideDocumentFormattingEdits(document) {
      const formatted = await formatDocument(exe, document.getText());
      if (formatted === document.getText()) {
        return [];
      }

      const fullRange = new vscode.Range(
        document.positionAt(0),
        document.positionAt(document.getText().length),
      );
      return [vscode.TextEdit.replace(fullRange, formatted)];
    },
  });
}

async function formatDocument(exe: string, text: string): Promise<string> {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "tya_format_"));
  const tempFile = path.join(tempDir, "Document.tya");
  try {
    fs.writeFileSync(tempFile, text);
    const { stdout } = await execFileAsync(exe, ["format", tempFile], {
      maxBuffer: 10 * 1024 * 1024,
    });
    return stdout;
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
}

function resolveExecutable(configured: string): string {
  if (configured && configured !== "tya") {
    return configured;
  }

  let best: { path: string; version: number[] } | undefined;
  const home = os.homedir();
  for (const candidate of [
    path.join(home, ".local", "bin", "tya"),
    path.join(home, ".local", "share", "mise", "shims", "tya"),
    path.join(home, ".asdf", "shims", "tya"),
    path.join(home, ".cargo", "bin", "tya"),
    path.join(home, "go", "bin", "tya"),
    "/opt/homebrew/bin/tya",
    "/usr/local/bin/tya",
    "/usr/bin/tya",
  ]) {
    if (!isExecutable(candidate)) {
      continue;
    }
    const version = executableVersion(candidate);
    if (!version) {
      if (!best) {
        best = { path: candidate, version: [] };
      }
      continue;
    }
    if (!best || compareVersions(version, best.version) > 0) {
      best = { path: candidate, version };
    }
  }

  return best?.path ?? "tya";
}

function isExecutable(filePath: string): boolean {
  try {
    fs.accessSync(filePath, fs.constants.X_OK);
    return true;
  } catch {
    return false;
  }
}

function executableVersion(filePath: string): number[] | undefined {
  try {
    const stdout = execFileSync(filePath, ["version"], {
      encoding: "utf8",
      timeout: 2000,
      maxBuffer: 1024 * 1024,
    });
    return parseVersion(stdout.trim());
  } catch {
    return undefined;
  }
}

function parseVersion(value: string): number[] | undefined {
  const match = value.match(/^(\d+)\.(\d+)\.(\d+)$/);
  if (!match) {
    return undefined;
  }
  return [Number(match[1]), Number(match[2]), Number(match[3])];
}

function compareVersions(left: number[], right: number[]): number {
  for (let i = 0; i < 3; i++) {
    const l = left[i] ?? -1;
    const r = right[i] ?? -1;
    if (l !== r) {
      return l - r;
    }
  }
  return 0;
}
