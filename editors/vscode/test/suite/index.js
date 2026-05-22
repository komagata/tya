const assert = require("assert");
const fs = require("fs");
const os = require("os");
const path = require("path");
const vscode = require("vscode");
const Mocha = require("mocha");

exports.run = function () {
  const mocha = new Mocha({ ui: "tdd", color: true, timeout: 20000 });
  mocha.suite.emit("pre-require", global, "tya-format-on-save.test.js", mocha);

  suite("Tya format on save", () => {
    setup(async () => {
      const extension = vscode.extensions.getExtension("komagata.tya");
      assert.ok(extension, "komagata.tya extension is available");
      await extension.activate();
      assert.strictEqual(extension.isActive, true);
    });

    test("contributes Go-style format-on-save defaults", () => {
      const config = vscode.workspace.getConfiguration("editor", { languageId: "tya" });
      assert.strictEqual(config.get("formatOnSave"), true);
      assert.strictEqual(config.get("defaultFormatter"), "komagata.tya");
    });

    test("formats all supported tya format cases on save through the extension formatter", async () => {
      const cases = [
        {
          name: "strings-and-containers",
          input: [
            "items = [1, 2,]",
            "user = { name: 'Ada', age: 20, }",
            "out = {}",
            "out['repo'] = user['name']",
            "print(out['repo'])",
            "",
          ].join("\n"),
          want: [
            "items = [1, 2]",
            "user = { name: \"Ada\", age: 20 }",
            "out = {}",
            "out[\"repo\"] = user[\"name\"]",
            "print(out[\"repo\"])",
            "",
          ].join("\n"),
        },
        {
          name: "single-quoted-string-escapes",
          input: "value = '{name} } \" \\\\ \\' \\n \\t \\r'\n",
          want: "value = \"{{name}} }} \\\" \\\\ ' \\n \\t \\r\"\n",
        },
        {
          name: "calls-and-lambdas",
          input: [
            "add = (a, b,) -> a + b",
            "print(add(1, 2,))",
            "class Box",
            "  static get = ->",
            "    return Self.wrap(self.value + super.value)",
            "",
          ].join("\n"),
          want: [
            "add = a, b -> a + b",
            "print(add(1, 2))",
            "class Box",
            "  static get = () ->",
            "    return Self.wrap(self.value + super.value)",
            "",
          ].join("\n"),
        },
        {
          name: "comments",
          input: [
            "# header line one",
            "# header line two",
            "",
            "# greet a user",
            "greet = name -> name",
            "x = 1  # initial value",
            "",
          ].join("\n"),
          want: [
            "# header line one",
            "# header line two",
            "",
            "# greet a user",
            "greet = name -> name",
            "x = 1  # initial value",
            "",
          ].join("\n"),
        },
        {
          name: "if-elseif-else",
          input: [
            "x = 1",
            "if x == 0",
            "  print(\"a\")",
            "elseif x == 1",
            "  print(\"b\")",
            "else",
            "  print(\"c\")",
            "",
          ].join("\n"),
          want: [
            "x = 1",
            "if x == 0",
            "  print(\"a\")",
            "elseif x == 1",
            "  print(\"b\")",
            "else",
            "  print(\"c\")",
            "",
          ].join("\n"),
        },
        {
          name: "try-catch-finally",
          input: [
            "try",
            "  print(\"try\")",
            "catch err",
            "  print(err)",
            "finally",
            "  print(\"done\")",
            "",
          ].join("\n"),
          want: [
            "try",
            "  print(\"try\")",
            "catch err",
            "  print(err)",
            "finally",
            "  print(\"done\")",
            "",
          ].join("\n"),
        },
        {
          name: "block-lambda",
          input: [
            "f = x ->",
            "  y = x + 1",
            "  return y",
            "print(f(2))",
            "",
          ].join("\n"),
          want: [
            "f = x ->",
            "  y = x + 1",
            "  return y",
            "print(f(2))",
            "",
          ].join("\n"),
        },
        {
          name: "imports",
          input: [
            "import zmod",
            "import string",
            "import file",
            "import mylib",
            "",
            "greet = name -> name",
            "x = 1",
            "",
          ].join("\n"),
          want: [
            "import file",
            "import string",
            "",
            "import mylib",
            "import zmod",
            "",
            "greet = name -> name",
            "x = 1",
            "",
          ].join("\n"),
        },
        {
          name: "long-string-newlines",
          input: "msg = \"line one of message\\nline two of message\\nline three of message and a bit more here\"\n",
          want: [
            "msg = \"\"\"",
            "  line one of message",
            "  line two of message",
            "  line three of message and a bit more here",
            "  \"\"\"",
            "",
          ].join("\n"),
        },
        {
          name: "long-string-without-whitespace",
          input: "url = \"https://example.com/path/to/very/long/resource/that/does/not/contain/whitespace\"\n",
          want: "url = \"https://example.com/path/to/very/long/resource/that/does/not/contain/whitespace\"\n",
        },
        {
          name: "long-dict",
          input: "user = { full_name: \"Alice Example\", role: \"administrator-level\", region: \"asia\" }\n",
          want: [
            "user =",
            "  full_name: \"Alice Example\"",
            "  role: \"administrator-level\"",
            "  region: \"asia\"",
            "",
          ].join("\n"),
        },
        {
          name: "binary-chain",
          input: "total = first_part_value + second_part_value + third_part_value + fourth_part_value\n",
          want: [
            "total = first_part_value",
            "  + second_part_value",
            "  + third_part_value",
            "  + fourth_part_value",
            "",
          ].join("\n"),
        },
        {
          name: "long-if-condition",
          input: [
            "if some_condition_value + another_part_value > threshold_limit and not exceptional_case_was_seen",
            "  process(a, b)",
            "",
          ].join("\n"),
          want: [
            "if (",
            "  some_condition_value + another_part_value > threshold_limit and not exceptional_case_was_seen",
            ")",
            "  process(a, b)",
            "",
          ].join("\n"),
        },
        {
          name: "long-lambda-body",
          input: "greet = recipient_name -> \"Hello, \" + recipient_name + \"! Welcome to the service.\"\n",
          want: [
            "greet = recipient_name ->",
            "  \"Hello, \" + recipient_name + \"! Welcome to the service.\"",
            "",
          ].join("\n"),
        },
        {
          name: "long-call",
          input: "result = compute_filtered_items(source_alpha, source_beta, source_gamma, source_delta)\n",
          want: [
            "result = compute_filtered_items(",
            "  source_alpha,",
            "  source_beta,",
            "  source_gamma,",
            "  source_delta",
            ")",
            "",
          ].join("\n"),
        },
        {
          name: "long-array",
          input: "items = [first_item_name, second_item_name, third_item_name, fourth_item_name, fifth_item_name]\n",
          want: [
            "items = [",
            "  first_item_name,",
            "  second_item_name,",
            "  third_item_name,",
            "  fourth_item_name,",
            "  fifth_item_name",
            "]",
            "",
          ].join("\n"),
        },
        {
          name: "class",
          input: [
            "class Dog",
            "  bark = ->",
            "    return \"woof\"",
            "",
          ].join("\n"),
          want: [
            "class Dog",
            "  bark = () ->",
            "    return \"woof\"",
            "",
          ].join("\n"),
        },
        {
          name: "match",
          input: [
            "match x",
            "  case 1",
            "    print(\"one\")",
            "  case _",
            "    print(\"other\")",
            "",
          ].join("\n"),
          want: [
            "match x",
            "  case 1",
            "    print(\"one\")",
            "  case _",
            "    print(\"other\")",
            "",
          ].join("\n"),
        },
      ];

      for (const tc of cases) {
        const file = path.join(os.tmpdir(), `tya-format-on-save-${process.pid}-${tc.name}.tya`);
        fs.writeFileSync(file, "");
        let document;
        try {
          document = await vscode.workspace.openTextDocument(file);
          await vscode.window.showTextDocument(document);
          assert.strictEqual(document.languageId, "tya");
          const edit = new vscode.WorkspaceEdit();
          edit.insert(document.uri, new vscode.Position(0, 0), tc.input);
          assert.strictEqual(await vscode.workspace.applyEdit(edit), true);
          await document.save();
          await waitForFile(file, tc.want);
          assert.strictEqual(fs.readFileSync(file, "utf8"), tc.want);
        } finally {
          if (document) {
            await vscode.window.showTextDocument(document);
            await vscode.commands.executeCommand("workbench.action.closeActiveEditor");
          }
          fs.rmSync(file, { force: true });
        }
      }
    });
  });

  return new Promise((resolve, reject) => {
    mocha.run((failures) => {
      if (failures > 0) {
        reject(new Error(`${failures} tests failed`));
      } else {
        resolve();
      }
    });
  });
};

async function waitForFile(file, want) {
  const deadline = Date.now() + 10000;
  while (Date.now() < deadline) {
    if (fs.readFileSync(file, "utf8") === want) {
      return;
    }
    await sleep(100);
  }
  throw new Error(`file was not formatted on save: ${file}`);
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
