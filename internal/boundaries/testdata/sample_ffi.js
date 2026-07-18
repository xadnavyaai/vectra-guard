// Sample Node file with FFI bindings.

const ffi = require('ffi-napi');
const ref = require('ref-napi');
const bindings = require('bindings');

function safeAdd(a, b) {
  return a + b;
}

module.exports = { safeAdd };
