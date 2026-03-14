# Self Tests

Tests in this directory run the tool against its own spec. The AASDD CLI is itself specified using AASDD, so these tests verify mutual consistency — the tool can parse, export, import, and verify the spec that defines it.

This is a distinct category from unit tests (single-ability contracts) and integration tests (spec scenarios). These are self-referential conformance checks.
