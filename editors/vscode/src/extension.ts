import * as vscode from "vscode";
import * as fs from "fs";
import * as os from "os";
import * as path from "path";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;

export function activate(_context: vscode.ExtensionContext): void {
  const config = vscode.workspace.getConfiguration("tya");
  const exe = resolveExecutable(config.get<string>("executable", "tya"));
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
