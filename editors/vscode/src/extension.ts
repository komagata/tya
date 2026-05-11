import * as vscode from "vscode";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;

export function activate(_context: vscode.ExtensionContext): void {
  const config = vscode.workspace.getConfiguration("tya");
  const exe = config.get<string>("executable", "tya");
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
