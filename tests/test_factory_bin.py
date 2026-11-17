"""Tests for the factory scripts' base-branch and PR-target rules.

Run from the repo root: python3 -m unittest discover -s tests
Nothing here calls GitHub or a model: gh and git config are replaced.
"""

import importlib.machinery
import importlib.util
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

BIN = Path(__file__).resolve().parents[1] / "factory" / "bin"
sys.path.insert(0, str(BIN))
sys.dont_write_bytecode = True

import factory_config  # noqa: E402


def load_script(name):
    loader = importlib.machinery.SourceFileLoader(name, str(BIN / name))
    module = importlib.util.module_from_spec(importlib.util.spec_from_loader(name, loader))
    loader.exec_module(module)
    return module


job = load_script("job")
BASE = job.BASE
REPO = "acme/widget"
URL = f"https://github.com/{REPO}/pull/7"


def a_job(**fields):
    return {"github_repo": REPO, "intake_base": None, "base_branch": None, "base_sha": None, **fields}


class PrTargetProblem(unittest.TestCase):
    def setUp(self):
        self.gh = mock.patch.object(job, "gh", side_effect=AssertionError("gh must not be called"))
        self.gh.start()
        self.addCleanup(self.gh.stop)

    def test_right_base_and_repo_is_fine(self):
        self.assertIsNone(job.pr_target_problem(a_job(), 7, URL, BASE))

    def test_other_base_is_named(self):
        why = job.pr_target_problem(a_job(), 7, URL, "main")
        self.assertIn("targets `main`", why)
        self.assertIn(f"`{BASE}`", why)

    def test_repo_case_differences_are_the_same_repo(self):
        self.assertIsNone(job.pr_target_problem(a_job(github_repo="ACME/Widget"), 7, URL, BASE))

    def test_url_that_isnt_a_github_pr_is_refused(self):
        self.assertIn("isn't in acme/widget", job.pr_target_problem(a_job(), 7, "not-a-url", BASE))


class SameRepo(unittest.TestCase):
    def test_renamed_origin_matches_the_current_name(self):
        with mock.patch.object(job, "gh", return_value="new-org/widget\n") as gh:
            self.assertTrue(job.same_repo("old-org/widget", "new-org/widget"))
        gh.assert_called_once()

    def test_another_repo_does_not_match(self):
        with mock.patch.object(job, "gh", return_value="acme/widget\n"):
            self.assertFalse(job.same_repo("acme/widget", "someone/fork"))

    def test_gh_failure_does_not_match(self):
        with mock.patch.object(job, "gh", side_effect=job.Refused("not found")):
            self.assertFalse(job.same_repo("gone/repo", "acme/widget"))


class OriginRepo(unittest.TestCase):
    def origin(self, url, gh_says=None):
        done = subprocess.CompletedProcess([], 0, stdout=url + "\n", stderr="")
        gh = mock.patch.object(job, "gh", return_value=gh_says or "", side_effect=None if gh_says else AssertionError)
        with mock.patch.object(job, "run", return_value=done), gh:
            return job.origin_repo()

    def test_standard_url_shapes(self):
        for url in ("git@github.com:acme/widget.git", "https://github.com/acme/widget",
                    "ssh://git@github.com/acme/widget.git", "https://github.com/acme/widget.git/"):
            self.assertEqual(self.origin(url), REPO, url)

    def test_other_shapes_go_through_gh(self):
        self.assertEqual(self.origin("gh-alias:acme/widget.git", gh_says="acme/widget\n"), REPO)


class Migrate(unittest.TestCase):
    def test_old_repo_field_is_renamed_without_lookups(self):
        with mock.patch.object(job, "origin_repo", side_effect=AssertionError("no lookup on load")):
            migrated = job.migrate({"repo": REPO})
        self.assertEqual(migrated["github_repo"], REPO)
        self.assertNotIn("repo", migrated)
        self.assertIsNone(migrated["intake_base"])

    def test_missing_repo_is_looked_up_only_when_needed(self):
        with mock.patch.object(job, "origin_repo", side_effect=AssertionError("no lookup on load")):
            migrated = job.migrate({})
        self.assertIsNone(migrated["github_repo"])
        with mock.patch.object(job, "origin_repo", return_value=REPO):
            self.assertEqual(job.github_repo(migrated), REPO)
        self.assertEqual(migrated["github_repo"], REPO)


class ConfigLoad(unittest.TestCase):
    def load(self, config):
        with tempfile.TemporaryDirectory() as d:
            path = Path(d) / "config.json"
            path.write_text(json.dumps(config))
            with mock.patch.object(factory_config, "PATH", path):
                return factory_config.load("job")

    def test_base_branch_is_returned(self):
        self.assertEqual(self.load({"base_branch": "develop"})["base_branch"], "develop")

    def test_missing_or_empty_base_branch_exits_with_a_message(self):
        for config in ({}, {"base_branch": ""}, {"base_branch": "  "}, {"base_branch": 3}):
            with self.assertRaises(SystemExit) as e:
                self.load(config)
            self.assertIn('needs "base_branch"', str(e.exception.code), config)


def git(cwd, *args):
    subprocess.run(["git", *args], cwd=cwd, check=True, capture_output=True)


class RemoveIntakeTree(unittest.TestCase):
    def setUp(self):
        self.root = Path(tempfile.mkdtemp())
        self.addCleanup(subprocess.run, ["rm", "-rf", str(self.root)])
        git(self.root, "init", "-q", "-b", "main")
        (self.root / "f").write_text("x")
        git(self.root, "add", "f")
        git(self.root, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-qm", "init")
        # run()'s default cwd is the real repo, bound when it was defined: send
        # every git call these tests cause to the scratch repo instead.
        real_run = job.run
        for patch in (mock.patch.object(job, "REPO", self.root),
                      mock.patch.object(job, "run", lambda cmd, cwd=None, check=True:
                                        real_run(cmd, cwd=self.root, check=check))):
            patch.start()
            self.addCleanup(patch.stop)
        self.j = {"id": "001-test"}
        self.path = job.intake_tree(self.j)

    def add_tree(self):
        git(self.root, "worktree", "add", "-q", "--detach", str(self.path), "HEAD")

    def registered(self):
        out = subprocess.run(["git", "worktree", "list"], cwd=self.root, capture_output=True, text=True).stdout
        return str(self.path) in out

    def test_registered_tree_is_removed(self):
        self.add_tree()
        job.remove_intake_tree(self.j)
        self.assertFalse(self.path.exists())
        self.assertFalse(self.registered())

    def test_registration_whose_directory_is_gone_is_cleared(self):
        self.add_tree()
        subprocess.run(["rm", "-rf", str(self.path)], check=True)
        job.remove_intake_tree(self.j)
        self.assertFalse(self.registered())
        self.add_tree()  # a new tree can take its place

    def test_unregistered_directory_is_deleted(self):
        self.path.mkdir(parents=True)
        (self.path / "junk").write_text("stale")
        job.remove_intake_tree(self.j)
        self.assertFalse(self.path.exists())

    def test_nothing_there_is_fine(self):
        job.remove_intake_tree(self.j)
        self.assertFalse(self.path.exists())


if __name__ == "__main__":
    unittest.main()
