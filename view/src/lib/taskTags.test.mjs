import assert from 'node:assert/strict';
import test from 'node:test';

import { TASK_TYPE_TONES, typePillClass, typeTone } from './taskTags.js';

test('known task types keep a stable highlighted tone', () => {
  assert.equal(typeTone('bug'), 'red');
  assert.equal(typeTone('Feature'), 'blue');
  assert.equal(typeTone('ui'), 'pink');
  assert.equal(typeTone('docs'), 'teal');
  assert.equal(typeTone('architecture'), 'indigo');
  assert.equal(typeTone('refactor'), 'cyan');
  assert.equal(typeTone('maintenance'), 'slate');
});

test('unknown task types still resolve to a palette tone', () => {
  const tone = typeTone('spike');
  assert.equal(TASK_TYPE_TONES.spike, undefined);
  assert.equal(typeTone('spike'), tone);
  assert.match(typePillClass('spike'), new RegExp(`pill-tone-${tone}`));
});
