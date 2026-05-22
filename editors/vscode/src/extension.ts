import * as vscode from "vscode";
import { execFile } from "child_process";
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
    if (isExecutable(candidate)) {
      return candidate;
    }
  }

  return "tya";
}

function isExecutable(filePath: string): boolean {
  try {
    fs.accessSync(filePath, fs.constants.X_OK);
    return true;
  } catch {
    return false;
  }
}
