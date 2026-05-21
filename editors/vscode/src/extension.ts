import * as vscode from "vscode";
import * as fs from "fs";
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

  for (const path of [
    "/opt/homebrew/bin/tya",
    "/usr/local/bin/tya",
    "/usr/bin/tya",
  ]) {
    if (fs.existsSync(path)) {
      return path;
    }
  }

  return "tya";
}
