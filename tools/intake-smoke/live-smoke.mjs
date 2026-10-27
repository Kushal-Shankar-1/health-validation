// SPDX-FileCopyrightText: Copyright (c) 2026, NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Isolated fork smoke test. Every affiliation record below is a synthetic fixture.
import assert from 'node:assert/strict';
import { classifyFromActions, createGitHubClient } from './actions-intake.mjs';

const repository = { id: 1427862585, full_name: 'Kushal-Shankar-1/health-validation' };
const prefix = '[intake smoke test]';
assert.equal(process.env.GITHUB_REPOSITORY, repository.full_name);
assert.equal(process.env.GITHUB_REF, 'refs/heads/test/intake-live-smoke-20261026');
const token = process.env.GITHUB_TOKEN;
assert.ok(token);
const issue = Number(process.env.SMOKE_ISSUE);
const pr = Number(process.env.SMOKE_PR);
assert.ok(Number.isSafeInteger(issue) && issue > 0);
assert.ok(Number.isSafeInteger(pr) && pr > 0 && issue !== pr);
const root = `https://api.github.com/repos/${repository.full_name}`;
async function api(path, method = 'GET', body) {
  const response = await fetch(root + path, {
    method, redirect: 'error', signal: AbortSignal.timeout(10000),
    headers: { Authorization: `Bearer ${token}`, Accept: 'application/vnd.github+json',
      'Content-Type': 'application/json', 'X-GitHub-Api-Version': '2022-11-28' },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!response.ok) throw new Error(`Sandbox API ${method} ${path} failed with HTTP ${response.status}`);
  return response.status === 204 ? null : response.json();
}
const client = createGitHubClient({ token });
let writes = 0;
const tracked = {
  readItem: (...args) => client.readItem(...args),
  addLabels: async (...args) => { writes++; return client.addLabels(...args); },
  removeLabel: async (...args) => { writes++; return client.removeLabel(...args); },
};
const now = new Date().toISOString();
const epoch = Date.parse(now);
const review = (user, affiliation) => ({ author_id: user.id, login: user.login, affiliation,
  reviewed_by: 'synthetic-smoke-test', evidence_ref: 'synthetic-fixture-only', revision: 'test-1',
  reviewed_at: new Date(epoch - 60000).toISOString(),
  expires_at: new Date(epoch + 3600000).toISOString() });
const managed = value => value.startsWith('affiliation/');
async function check(kind, number, evidence, expected, manual = false) {
  const raw = await client.readItem(kind, number);
  assert.ok(raw.title.startsWith(prefix));
  const beforeLabels = raw.labels.map(label => label.name).filter(label => !managed(label)).sort();
  const beforeAssignees = raw.assignees.map(user => user.id).sort();
  const payload = manual ? { repository, inputs: { kind, number: String(number) } }
    : { action: 'opened', repository, [kind === 'pr' ? 'pull_request' : 'issue']: raw };
  const result = await classifyFromActions({ eventName: manual ? 'workflow_dispatch'
    : kind === 'pr' ? 'pull_request_target' : 'issues', payload, evaluatedAt: now,
    reviewedDecisions: evidence }, tracked);
  assert.equal(result.status, 'classified');
  assert.equal(result.decision.affiliation, expected);
  const after = await client.readItem(kind, number);
  assert.deepEqual(after.labels.map(label => label.name).filter(managed), [`affiliation/${expected}`]);
  assert.deepEqual(after.labels.map(label => label.name).filter(label => !managed(label)).sort(), beforeLabels);
  assert.deepEqual(after.assignees.map(user => user.id).sort(), beforeAssignees);
  console.log(`PASS ${kind} ${manual ? 'manual payload' : 'opened payload'}: ${expected}; unrelated metadata preserved`);
}
const originals = [];
const createdLabels = [];
let botIssue;
let stage = 'validate fixtures';
let failure;
try {
  // Validate every pre-created target before changing any labels.
  for (const [kind, number] of [['issue', issue], ['pr', pr]]) {
    const raw = await client.readItem(kind, number);
    assert.ok(raw.title.startsWith(prefix));
    assert.equal(raw.user.id, 141350084);
    assert.equal(raw.state, 'open');
    assert.ok(raw.labels.some(label => label.name === 'bug'));
    assert.ok(raw.assignees.some(user => user.id === 141350084));
    if (kind === 'pr') assert.equal(raw.draft, true);
    originals.push({ kind, number, labels: raw.labels.map(label => label.name) });
  }
  stage = 'create sandbox labels';
  for (const name of ['nvidia', 'external', 'unverified', 'bot'].map(value => `affiliation/${value}`).concat('priority/P2')) {
    const labels = await api('/labels?per_page=100');
    if (!labels.some(label => label.name === name)) {
      await api('/labels', 'POST', {
        name, color: 'ededed', description: 'Synthetic sandbox classification test label',
      });
      createdLabels.push(name);
    }
  }
  stage = 'seed priority labels';
  for (const { number } of originals) {
    await api(`/issues/${number}/labels`, 'POST', { labels: ['priority/P2'] });
  }
  stage = 'classify human issue and draft PR';
  for (const { kind, number } of originals) {
    const raw = await client.readItem(kind, number);
    await check(kind, number, [], 'unverified');
    await check(kind, number, [review(raw.user, 'nvidia')], 'nvidia');
    await check(kind, number, [review(raw.user, 'external')], 'external');
    const beforeRepeat = writes;
    await check(kind, number, [review(raw.user, 'external')], 'external');
    assert.equal(writes, beforeRepeat);
    console.log(`PASS ${kind}: repeat classification made no writes`);
    const expired = { ...review(raw.user, 'nvidia'),
      reviewed_at: new Date(epoch - 7200000).toISOString(),
      expires_at: new Date(epoch - 3600000).toISOString() };
    await check(kind, number, [expired], 'unverified');
    await check(kind, number, [review(raw.user, 'external')], 'external', true);
  }
  stage = 'classify bot issue';
  botIssue = await api('/issues', 'POST', {
    title: `${prefix} bot-created fixture`,
    body: 'Temporary automation test in a personal fork. No support request or actual affiliation evidence.',
  });
  assert.equal(botIssue.user.type, 'Bot');
  await check('issue', botIssue.number, [], 'bot');
  const count = writes;
  await assert.rejects(classifyFromActions({ eventName: 'workflow_dispatch',
    payload: { repository: { ...repository, id: 1 }, inputs: { kind: 'issue', number: String(issue) } },
    evaluatedAt: now }, tracked), /scope/);
  assert.equal(writes, count);
  console.log('PASS wrong repository: rejected before writes');
  console.log('All live API smoke checks passed. Synthetic evidence does not establish anyone\'s affiliation.');
} catch (error) {
  failure = error;
  console.error(`Sandbox failure during ${stage}: ${error.message}`);
} finally {
  // Remove synthetic affiliation assertions even if a test fails.
  const cleanup = originals.map(({ number, labels }) =>
    api(`/issues/${number}/labels`, 'PUT', { labels }));
  if (botIssue) {
    cleanup.push(api(`/issues/${botIssue.number}/labels`, 'PUT', { labels: [] }));
    cleanup.push(api(`/issues/${botIssue.number}`, 'PATCH', { state: 'closed' }));
  }
  const restored = await Promise.allSettled(cleanup);
  const labels = await Promise.allSettled(createdLabels.map(name =>
    api(`/labels/${encodeURIComponent(name)}`, 'DELETE')));
  const failures = [...restored, ...labels].filter(result => result.status === 'rejected');
  for (const result of failures) console.error(`Cleanup failure: ${result.reason.message}`);
  if (failure) throw failure;
  assert.equal(failures.length, 0, 'Sandbox cleanup incomplete; supervisor reconciliation required');
}
