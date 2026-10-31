import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { findUnreachableDefinitions, isNamedIn, listDefinitions, verifyProtocolHoldsOnlyHostContracts } from "./protocol.mjs";

const PROTO = `syntax = "proto3";
service VaultService {
  rpc Open(OpenRequest) returns (OpenResponse);
}
message OpenRequest { Mode mode = 1; }
message OpenResponse { string path = 1; }
enum Mode { MODE_UNSPECIFIED = 0; }
service TranscriptService {
  rpc Get(GetTranscriptRequest) returns (Transcript);
}
message GetTranscriptRequest { string id = 1; }
message Transcript { repeated Segment segments = 1; message Segment { string text = 1; } }
message Segment { string text = 1; }
`;

const definitions = () => listDefinitions([{ file: "a.proto", code: PROTO }]);

describe("the definitions of a proto source", () => {
  it("are the top-level services, messages and enums", () => {
    assert.deepEqual(
      [...definitions().keys()],
      ["VaultService", "OpenRequest", "OpenResponse", "Mode", "TranscriptService", "GetTranscriptRequest", "Transcript", "Segment"],
    );
  });

  it("keep the line of their declaration and their body", () => {
    const { line, body } = definitions().get("OpenResponse");
    assert.equal(line, 6);
    assert.match(body, /string path/);
  });
});

describe("a name stated in a source", () => {
  it("is found as an identifier, a generated type, a client and a handler constructor", () => {
    for (const text of ["numenv1.VaultService", "const c: VaultServiceClient", "NewVaultServiceHandler(x)", "VaultService_Open"]) {
      assert.ok(isNamedIn("VaultService", text), text);
    }
  });

  it("is not found inside a longer name", () => {
    assert.equal(isNamedIn("Mode", "const nodeModeX = 1; Modest"), false);
  });
});

describe("a definition that no host source uses", () => {
  it("is reported with its file and line, with everything only it uses", () => {
    const found = findUnreachableDefinitions(definitions(), "client.Open(numenv1.OpenRequest{})");
    assert.deepEqual(
      found.map(({ symbol }) => symbol.split(" ").slice(0, 2).join(" ")),
      ["service VaultService", "message OpenResponse", "service TranscriptService", "message GetTranscriptRequest", "message Transcript", "message Segment"],
    );
    assert.equal(found[0].file, "a.proto");
  });

  it("is reachable through a definition that a host source uses", () => {
    const found = findUnreachableDefinitions(definitions(), "NewVaultServiceHandler(h); NewTranscriptServiceHandler(h)");
    assert.deepEqual(found, []);
  });

  it("is reported when only a plugin source states it", () => {
    const found = findUnreachableDefinitions(definitions(), "NewVaultServiceHandler(h)");
    assert.deepEqual(
      found.map(({ symbol }) => symbol.split(" ")[1]),
      ["TranscriptService", "GetTranscriptRequest", "Transcript", "Segment"],
    );
  });
});

describe("the live protocol", () => {
  it("holds only contracts that the host uses", () => {
    assert.doesNotThrow(verifyProtocolHoldsOnlyHostContracts);
  });
});
