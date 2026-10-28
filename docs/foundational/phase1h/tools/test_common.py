import importlib.util, json, pathlib, tempfile, unittest

spec = importlib.util.spec_from_file_location(
    "common", pathlib.Path(__file__).with_name("common.py")
)
c = importlib.util.module_from_spec(spec)


class Contracts(unittest.TestCase):
    def test_source_detects_new_and_changed_files(self):
        with tempfile.TemporaryDirectory() as d:
            root = pathlib.Path(d)
            (root / "author").mkdir()
            p = root / "author/a.go"
            p.write_text("a")
            first = c.inventory(root)
            p.write_text("b")
            self.assertNotEqual(first, c.inventory(root))
            p.write_text("a")
            (root / "author/new.go").write_text("new")
            self.assertNotEqual(first, c.inventory(root))

    def test_governing_and_manuscript_inputs(self):
        with tempfile.TemporaryDirectory() as d:
            root = pathlib.Path(d)
            (root / "paper").mkdir()
            (root / "AGENTS.md").write_text("governing")
            (root / "paper/model.tex").write_text("model")
            files = c.inventory(root)
            self.assertIn("AGENTS.md", files)
            self.assertIn("paper/model.tex", files)
            (root / "AGENTS.md").write_text("new governing")
            self.assertNotEqual(files, c.inventory(root))

    def test_adaptive_marker_and_order(self):
        raw = "".join(
            json.dumps(x) + "\n"
            for x in [
                {
                    "Action": "output",
                    "Package": "p",
                    "Test": "T",
                    "Output": '    t.go:1: ADAPTIVE {"kind":"cut"}\n',
                },
                {"Action": "pass", "Package": "p", "Test": "T"},
            ]
        ).encode()
        r = c.parse(raw, "source", "focused")
        self.assertEqual(r["records"][0]["Marker"], "ADAPTIVE")
        self.assertEqual(r["records"][0]["SourceID"], "source")
        self.assertEqual(r["passes"], ["p/T"])


if __name__ == "__main__":
    spec.loader.exec_module(c)
    unittest.main()
