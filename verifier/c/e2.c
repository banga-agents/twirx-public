#include "e2.h"

#include <string.h>

#define TW_RESULT_FIELDS 17U
#define TW_PROVENANCE_FIELDS 12U

typedef struct tw_e2_parser {
    const uint8_t *data;
    size_t length;
    size_t offset;
} tw_e2_parser;

static bool take(tw_e2_parser *parser, size_t count, const uint8_t **out) {
    if (count > parser->length - parser->offset) {
        return false;
    }
    *out = &parser->data[parser->offset];
    parser->offset += count;
    return true;
}

static bool head(tw_e2_parser *parser, uint8_t *major, uint64_t *value) {
    const uint8_t *bytes = NULL;
    if (!take(parser, 1U, &bytes)) {
        return false;
    }
    const uint8_t initial = bytes[0];
    *major = (uint8_t)(initial >> 5U);
    const uint8_t additional = (uint8_t)(initial & UINT8_C(0x1f));
    if (additional < 24U) {
        *value = (uint64_t)additional;
        return true;
    }
    size_t count = 0U;
    uint64_t minimum = UINT64_C(0);
    switch (additional) {
        case 24U: count = 1U; minimum = UINT64_C(24); break;
        case 25U: count = 2U; minimum = UINT64_C(256); break;
        case 26U: count = 4U; minimum = UINT64_C(65536); break;
        case 27U: count = 8U; minimum = UINT64_C(4294967296); break;
        default: return false;
    }
    if (!take(parser, count, &bytes)) {
        return false;
    }
    *value = UINT64_C(0);
    for (size_t i = 0U; i < count; ++i) {
        *value = (*value << 8U) | (uint64_t)bytes[i];
    }
    return *value >= minimum;
}

static bool array(tw_e2_parser *parser, uint64_t *length) {
    uint8_t major = 0U;
    return head(parser, &major, length) && major == 4U;
}

static bool uint_value(tw_e2_parser *parser, uint64_t *value) {
    uint8_t major = 0U;
    return head(parser, &major, value) && major == 0U;
}

static bool valid_utf8(const uint8_t *bytes, size_t length) {
    size_t offset = 0U;
    while (offset < length) {
        const uint8_t first = bytes[offset];
        if (first <= UINT8_C(0x7f)) {
            offset++;
        } else if (first >= UINT8_C(0xc2) && first <= UINT8_C(0xdf)) {
            if (offset + 1U >= length || bytes[offset + 1U] < UINT8_C(0x80) || bytes[offset + 1U] > UINT8_C(0xbf)) return false;
            offset += 2U;
        } else if (first >= UINT8_C(0xe0) && first <= UINT8_C(0xef)) {
            if (offset + 2U >= length) return false;
            const uint8_t second = bytes[offset + 1U];
            const uint8_t third = bytes[offset + 2U];
            if (third < UINT8_C(0x80) || third > UINT8_C(0xbf)) return false;
            if ((first == UINT8_C(0xe0) && (second < UINT8_C(0xa0) || second > UINT8_C(0xbf))) ||
                (first == UINT8_C(0xed) && (second < UINT8_C(0x80) || second > UINT8_C(0x9f))) ||
                (first != UINT8_C(0xe0) && first != UINT8_C(0xed) && (second < UINT8_C(0x80) || second > UINT8_C(0xbf)))) return false;
            offset += 3U;
        } else if (first >= UINT8_C(0xf0) && first <= UINT8_C(0xf4)) {
            if (offset + 3U >= length) return false;
            const uint8_t second = bytes[offset + 1U];
            const uint8_t third = bytes[offset + 2U];
            const uint8_t fourth = bytes[offset + 3U];
            if (third < UINT8_C(0x80) || third > UINT8_C(0xbf) || fourth < UINT8_C(0x80) || fourth > UINT8_C(0xbf)) return false;
            if ((first == UINT8_C(0xf0) && (second < UINT8_C(0x90) || second > UINT8_C(0xbf))) ||
                (first == UINT8_C(0xf4) && (second < UINT8_C(0x80) || second > UINT8_C(0x8f))) ||
                (first != UINT8_C(0xf0) && first != UINT8_C(0xf4) && (second < UINT8_C(0x80) || second > UINT8_C(0xbf)))) return false;
            offset += 4U;
        } else {
            return false;
        }
    }
    return true;
}

static bool text(tw_e2_parser *parser, char *out, size_t capacity, bool allow_empty) {
    uint8_t major = 0U;
    uint64_t length64 = UINT64_C(0);
    if (!head(parser, &major, &length64) || major != 3U || length64 >= (uint64_t)capacity || (!allow_empty && length64 == UINT64_C(0))) return false;
    const size_t length = (size_t)length64;
    const uint8_t *bytes = NULL;
    if (!take(parser, length, &bytes) || memchr(bytes, '\0', length) != NULL || !valid_utf8(bytes, length)) return false;
    memcpy(out, bytes, length);
    out[length] = '\0';
    return true;
}

static bool bytes32(tw_e2_parser *parser, uint8_t out[32U]) {
    uint8_t major = 0U;
    uint64_t length = UINT64_C(0);
    const uint8_t *bytes = NULL;
    if (!head(parser, &major, &length) || major != 2U || length != UINT64_C(32) || !take(parser, 32U, &bytes)) return false;
    memcpy(out, bytes, 32U);
    return true;
}

static bool literal(tw_e2_parser *parser, const char *expected) {
    char value[128U];
    return text(parser, value, sizeof(value), false) && strcmp(value, expected) == 0;
}

static bool parse_field(tw_e2_parser *parser, char ids[TW_E2_MAX_FIELDS][257U], size_t index, bool *unresolved) {
    uint64_t count = UINT64_C(0);
    char status[16U];
    char scratch[TW_E2_MAX_TEXT_BYTES + 1U];
    char native_lexical[(256U * 1024U) + 1U];
    char semantic_lexical[(256U * 1024U) + 1U];
    uint64_t native_present = UINT64_C(0);
    uint64_t semantic_present = UINT64_C(0);
    if (!array(parser, &count) || count != TW_PROVENANCE_FIELDS ||
        !text(parser, ids[index], 257U, false) || !text(parser, status, sizeof(status), false) ||
        !text(parser, scratch, sizeof(scratch), false) || !text(parser, scratch, sizeof(scratch), false) ||
        !uint_value(parser, &native_present) || native_present > UINT64_C(1) ||
        !text(parser, native_lexical, sizeof(native_lexical), true) ||
        !text(parser, scratch, sizeof(scratch), false) || !text(parser, scratch, sizeof(scratch), false) ||
        !uint_value(parser, &semantic_present) || semantic_present > UINT64_C(1) ||
        !text(parser, semantic_lexical, sizeof(semantic_lexical), true) || !array(parser, &count) || count > UINT64_C(16)) return false;
    for (uint64_t i = UINT64_C(0); i < count; ++i) {
        if (!text(parser, scratch, sizeof(scratch), false)) return false;
    }
    if (!text(parser, scratch, sizeof(scratch), false)) return false;
    for (size_t i = 0U; i < index; ++i) {
        if (strcmp(ids[i], ids[index]) == 0) return false;
    }
    if (strcmp(status, "resolved") == 0) {
        return native_present == UINT64_C(1) && semantic_present == UINT64_C(1);
    }
    if (strcmp(status, "unresolved") == 0) {
        *unresolved = true;
        return native_present == UINT64_C(0) && semantic_present == UINT64_C(0) && native_lexical[0] == '\0' && semantic_lexical[0] == '\0';
    }
    return false;
}

bool tw_e2_parse_result(const uint8_t *data, size_t length, tw_e2_result *result) {
    if (data == NULL || result == NULL || length == 0U || length > TW_E2_MAX_RESULT_BYTES) return false;
    tw_e2_parser parser = {.data=data, .length=length, .offset=0U};
    uint64_t count = UINT64_C(0);
    char version[32U];
    char effect[16U];
    char status[16U];
    if (!array(&parser, &count) || count != TW_RESULT_FIELDS || !text(&parser, version, sizeof(version), false) || strcmp(version, "tw.result/0.2") != 0 ||
        !text(&parser, result->invocation_id, sizeof(result->invocation_id), false) ||
        !text(&parser, result->origin_id, sizeof(result->origin_id), false) || !text(&parser, result->origin_version, sizeof(result->origin_version), false) ||
        !text(&parser, result->operation_id, sizeof(result->operation_id), false) || !text(&parser, result->operation_version, sizeof(result->operation_version), false) ||
        !text(&parser, effect, sizeof(effect), false) || strcmp(effect, "read") != 0 || !text(&parser, status, sizeof(status), false) ||
        (strcmp(status, "resolved") != 0 && strcmp(status, "partial") != 0) || !text(&parser, result->observed_at, sizeof(result->observed_at), false) ||
        !bytes32(&parser, result->input_digest) || !bytes32(&parser, result->observation_digest) || !bytes32(&parser, result->transport_digest) ||
        !bytes32(&parser, result->adapter_digest) || !bytes32(&parser, result->contract_digest) || !bytes32(&parser, result->semantic_closure_digest) ||
        !array(&parser, &count) || count == UINT64_C(0) || count > TW_E2_MAX_FIELDS) return false;
    result->field_count = (size_t)count;
    char ids[TW_E2_MAX_FIELDS][257U];
    bool unresolved = false;
    for (size_t i = 0U; i < result->field_count; ++i) {
        if (!parse_field(&parser, ids, i, &unresolved)) return false;
    }
    if ((unresolved && strcmp(status, "partial") != 0) || (!unresolved && strcmp(status, "resolved") != 0)) return false;
    if (!array(&parser, &count) || count != UINT64_C(0)) return false;
    return parser.offset == parser.length;
}

static bool safe_name(const char *name) {
    return name[0] != '\0' && strcmp(name, ".") != 0 && strcmp(name, "..") != 0 && strchr(name, '/') == NULL && strchr(name, '\\') == NULL;
}

static bool hex_digit(char value, uint8_t *out) {
    if (value >= '0' && value <= '9') { *out = (uint8_t)(value - '0'); return true; }
    if (value >= 'a' && value <= 'f') { *out = (uint8_t)(value - 'a' + 10); return true; }
    return false;
}

static bool digest_reference(const char *value, uint8_t out[32U]) {
    if (strlen(value) != 71U || memcmp(value, "sha256:", 7U) != 0) return false;
    for (size_t i = 0U; i < 32U; ++i) {
        uint8_t high = 0U;
        uint8_t low = 0U;
        if (!hex_digit(value[7U + i * 2U], &high) || !hex_digit(value[8U + i * 2U], &low)) return false;
        out[i] = (uint8_t)((high << 4U) | low);
    }
    return true;
}

const tw_e2_entry *tw_e2_find_entry(const tw_e2_manifest *manifest, const char *name) {
    for (size_t i = 0U; i < manifest->entry_count; ++i) {
        if (strcmp(manifest->entries[i].name, name) == 0) return &manifest->entries[i];
    }
    return NULL;
}

bool tw_e2_parse_manifest(const uint8_t *data, size_t length, tw_e2_manifest *manifest) {
    if (data == NULL || manifest == NULL || length == 0U || length > TW_E2_MAX_MANIFEST_BYTES) return false;
    tw_e2_parser parser = {.data=data, .length=length, .offset=0U};
    uint64_t count = UINT64_C(0);
    char result_id[72U];
    if (!array(&parser, &count) || count != UINT64_C(3) || !literal(&parser, "tw.bundle-manifest/0.1") ||
        !text(&parser, result_id, sizeof(result_id), false) || !digest_reference(result_id, manifest->result_digest) ||
        !array(&parser, &count) || count == UINT64_C(0) || count > TW_E2_MAX_ARTIFACTS) return false;
    manifest->entry_count = (size_t)count;
    for (size_t i = 0U; i < manifest->entry_count; ++i) {
        tw_e2_entry *entry = &manifest->entries[i];
        if (!array(&parser, &count) || count != UINT64_C(3) || !text(&parser, entry->name, sizeof(entry->name), false) ||
            !safe_name(entry->name) || !bytes32(&parser, entry->digest) || !uint_value(&parser, &entry->size) ||
            entry->size == UINT64_C(0) || entry->size > TW_E2_MAX_ARTIFACT_BYTES) return false;
        if (i > 0U && strcmp(manifest->entries[i - 1U].name, entry->name) >= 0) return false;
        if (strcmp(entry->name, "manifest.cbor") == 0) return false;
    }
    static const char *required[] = {"adapter.cbor", "contract.cbor", "input.cbor", "observation.cbor", "representation.body", "result.cbor", "semantic-closure.cbor", "transcript.json", "transport.cbor"};
    for (size_t i = 0U; i < sizeof(required) / sizeof(required[0]); ++i) {
        if (tw_e2_find_entry(manifest, required[i]) == NULL) return false;
    }
    const tw_e2_entry *result_entry = tw_e2_find_entry(manifest, "result.cbor");
    return result_entry != NULL && memcmp(result_entry->digest, manifest->result_digest, 32U) == 0 && parser.offset == parser.length;
}

bool tw_e2_parse_semantic_closure(const uint8_t *data, size_t length) {
    if (data == NULL || length == 0U || length > TW_E2_MAX_ARTIFACT_BYTES) return false;
    tw_e2_parser parser = {.data=data, .length=length, .offset=0U};
    uint64_t count = UINT64_C(0);
    char previous[TW_E2_MAX_TEXT_BYTES + 1U] = "";
    char current[TW_E2_MAX_TEXT_BYTES + 1U];
    if (!array(&parser, &count) || count != UINT64_C(2) || !literal(&parser, "tw.semantic-closure/0.1") ||
        !array(&parser, &count) || count == UINT64_C(0) || count > UINT64_C(32)) return false;
    for (uint64_t i = UINT64_C(0); i < count; ++i) {
        if (!text(&parser, current, sizeof(current), false) || (i > UINT64_C(0) && strcmp(previous, current) >= 0)) return false;
        (void)strcpy(previous, current);
    }
    return parser.offset == parser.length;
}
