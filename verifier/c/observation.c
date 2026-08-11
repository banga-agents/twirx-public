#include "observation.h"

#include <string.h>

#define TW_FIELD_COUNT 11U

typedef struct tw_parser {
    const uint8_t *data;
    size_t length;
    size_t offset;
} tw_parser;

static bool take(tw_parser *parser, size_t count, const uint8_t **out) {
    if (count > parser->length - parser->offset) {
        return false;
    }
    *out = &parser->data[parser->offset];
    parser->offset += count;
    return true;
}

static bool read_head(tw_parser *parser, uint8_t *major, uint64_t *value) {
    const uint8_t *bytes = NULL;
    if (!take(parser, 1U, &bytes)) {
        return false;
    }
    const uint8_t initial = bytes[0];
    *major = (uint8_t)(initial >> 5U);
    const uint8_t ai = (uint8_t)(initial & UINT8_C(0x1f));
    if (ai < 24U) {
        *value = (uint64_t)ai;
        return true;
    }
    if (ai == 24U) {
        if (!take(parser, 1U, &bytes)) {
            return false;
        }
        *value = (uint64_t)bytes[0];
        return *value >= UINT64_C(24);
    }
    if (ai == 25U) {
        if (!take(parser, 2U, &bytes)) {
            return false;
        }
        *value = ((uint64_t)bytes[0] << 8U) | (uint64_t)bytes[1];
        return *value > UINT64_C(0xff);
    }
    if (ai == 26U) {
        if (!take(parser, 4U, &bytes)) {
            return false;
        }
        *value = ((uint64_t)bytes[0] << 24U) |
                 ((uint64_t)bytes[1] << 16U) |
                 ((uint64_t)bytes[2] << 8U) |
                 (uint64_t)bytes[3];
        return *value > UINT64_C(0xffff);
    }
    if (ai == 27U) {
        if (!take(parser, 8U, &bytes)) {
            return false;
        }
        *value = UINT64_C(0);
        for (size_t i = 0U; i < 8U; ++i) {
            *value = (*value << 8U) | (uint64_t)bytes[i];
        }
        return *value > UINT64_C(0xffffffff);
    }
    return false;
}

static bool read_uint(tw_parser *parser, uint64_t *value) {
    uint8_t major = 0U;
    return read_head(parser, &major, value) && major == 0U;
}

static bool read_array(tw_parser *parser, uint64_t *length) {
    uint8_t major = 0U;
    return read_head(parser, &major, length) && major == 4U;
}

static bool valid_utf8(const uint8_t *bytes, size_t length) {
    size_t offset = 0U;
    while (offset < length) {
        const uint8_t first = bytes[offset];
        if (first <= UINT8_C(0x7f)) {
            offset++;
            continue;
        }
        if (first >= UINT8_C(0xc2) && first <= UINT8_C(0xdf)) {
            if (offset + 1U >= length || bytes[offset + 1U] < UINT8_C(0x80) || bytes[offset + 1U] > UINT8_C(0xbf)) {
                return false;
            }
            offset += 2U;
            continue;
        }
        if (first >= UINT8_C(0xe0) && first <= UINT8_C(0xef)) {
            if (offset + 2U >= length) {
                return false;
            }
            const uint8_t second = bytes[offset + 1U];
            const uint8_t third = bytes[offset + 2U];
            if (third < UINT8_C(0x80) || third > UINT8_C(0xbf)) {
                return false;
            }
            if ((first == UINT8_C(0xe0) && (second < UINT8_C(0xa0) || second > UINT8_C(0xbf))) ||
                (first == UINT8_C(0xed) && (second < UINT8_C(0x80) || second > UINT8_C(0x9f))) ||
                ((first != UINT8_C(0xe0) && first != UINT8_C(0xed)) && (second < UINT8_C(0x80) || second > UINT8_C(0xbf)))) {
                return false;
            }
            offset += 3U;
            continue;
        }
        if (first >= UINT8_C(0xf0) && first <= UINT8_C(0xf4)) {
            if (offset + 3U >= length) {
                return false;
            }
            const uint8_t second = bytes[offset + 1U];
            const uint8_t third = bytes[offset + 2U];
            const uint8_t fourth = bytes[offset + 3U];
            if (third < UINT8_C(0x80) || third > UINT8_C(0xbf) || fourth < UINT8_C(0x80) || fourth > UINT8_C(0xbf)) {
                return false;
            }
            if ((first == UINT8_C(0xf0) && (second < UINT8_C(0x90) || second > UINT8_C(0xbf))) ||
                (first == UINT8_C(0xf4) && (second < UINT8_C(0x80) || second > UINT8_C(0x8f))) ||
                ((first != UINT8_C(0xf0) && first != UINT8_C(0xf4)) && (second < UINT8_C(0x80) || second > UINT8_C(0xbf)))) {
                return false;
            }
            offset += 4U;
            continue;
        }
        return false;
    }
    return true;
}

static bool read_text(tw_parser *parser, char *out, size_t capacity) {
    uint8_t major = 0U;
    uint64_t length64 = UINT64_C(0);
    if (!read_head(parser, &major, &length64) || major != 3U || length64 == UINT64_C(0) || length64 >= (uint64_t)capacity) {
        return false;
    }
    const size_t length = (size_t)length64;
    const uint8_t *bytes = NULL;
    if (!take(parser, length, &bytes) || memchr(bytes, '\0', length) != NULL || !valid_utf8(bytes, length)) {
        return false;
    }
    memcpy(out, bytes, length);
    out[length] = '\0';
    return true;
}

static bool read_bytes_exact(tw_parser *parser, uint8_t *out, size_t expected) {
    uint8_t major = 0U;
    uint64_t length64 = UINT64_C(0);
    if (!read_head(parser, &major, &length64) || major != 2U || length64 != (uint64_t)expected) {
        return false;
    }
    const uint8_t *bytes = NULL;
    if (!take(parser, expected, &bytes)) {
        return false;
    }
    memcpy(out, bytes, expected);
    return true;
}

static unsigned int decimal2(const char *value) {
    return (unsigned int)(value[0] - '0') * 10U + (unsigned int)(value[1] - '0');
}

static unsigned int decimal4(const char *value) {
    return (unsigned int)(value[0] - '0') * 1000U + (unsigned int)(value[1] - '0') * 100U +
           (unsigned int)(value[2] - '0') * 10U + (unsigned int)(value[3] - '0');
}

static bool digits(const char *value, size_t count) {
    for (size_t i = 0U; i < count; ++i) {
        if (value[i] < '0' || value[i] > '9') {
            return false;
        }
    }
    return true;
}

static bool leap_year(unsigned int year) {
    return (year % 4U == 0U && year % 100U != 0U) || year % 400U == 0U;
}

static bool canonical_retrieved_at(const char *value) {
    const size_t length = strlen(value);
    if (length < 20U || length > TW_MAX_RETRIEVED_AT_BYTES || value[length - 1U] != 'Z' ||
        value[4] != '-' || value[7] != '-' || value[10] != 'T' || value[13] != ':' || value[16] != ':' ||
        !digits(value, 4U) || !digits(&value[5], 2U) || !digits(&value[8], 2U) ||
        !digits(&value[11], 2U) || !digits(&value[14], 2U) || !digits(&value[17], 2U)) {
        return false;
    }
    const unsigned int year = decimal4(value);
    const unsigned int month = decimal2(&value[5]);
    const unsigned int day = decimal2(&value[8]);
    const unsigned int hour = decimal2(&value[11]);
    const unsigned int minute = decimal2(&value[14]);
    const unsigned int second = decimal2(&value[17]);
    static const unsigned int month_days[12] = {31U, 28U, 31U, 30U, 31U, 30U, 31U, 31U, 30U, 31U, 30U, 31U};
    if (month < 1U || month > 12U || hour > 23U || minute > 59U || second > 59U) {
        return false;
    }
    unsigned int maximum_day = month_days[month - 1U];
    if (month == 2U && leap_year(year)) {
        maximum_day = 29U;
    }
    if (day < 1U || day > maximum_day) {
        return false;
    }
    if (length == 20U) {
        return true;
    }
    const size_t fraction_length = length - 21U;
    return value[19] == '.' && fraction_length >= 1U && fraction_length <= 9U &&
           digits(&value[20], fraction_length) && value[length - 2U] != '0';
}

bool tw_parse_observation(const uint8_t *data, size_t length, tw_observation *observation) {
    if (data == NULL || observation == NULL || length == 0U || length > TW_MAX_ENVELOPE_BYTES) {
        return false;
    }
    tw_parser parser = {.data = data, .length = length, .offset = 0U};
    uint64_t fields = UINT64_C(0);
    if (!read_array(&parser, &fields) || fields != TW_FIELD_COUNT) {
        return false;
    }
    if (!read_uint(&parser, &observation->version) || observation->version != UINT64_C(1)) {
        return false;
    }
    if (!read_text(&parser, observation->request_url, sizeof(observation->request_url)) ||
        !read_text(&parser, observation->final_url, sizeof(observation->final_url)) ||
        !read_text(&parser, observation->method, sizeof(observation->method)) ||
        !read_uint(&parser, &observation->status) ||
        !read_text(&parser, observation->media_type, sizeof(observation->media_type)) ||
        !read_text(&parser, observation->retrieved_at, sizeof(observation->retrieved_at)) ||
        !read_bytes_exact(&parser, observation->body_hash, sizeof(observation->body_hash)) ||
        !read_uint(&parser, &observation->body_size) ||
        !read_text(&parser, observation->policy_id, sizeof(observation->policy_id)) ||
        !read_text(&parser, observation->observer_id, sizeof(observation->observer_id))) {
        return false;
    }
    if (strcmp(observation->method, "GET") != 0 || observation->status < UINT64_C(100) ||
        observation->status > UINT64_C(599) || observation->body_size > TW_MAX_BODY_BYTES ||
        !canonical_retrieved_at(observation->retrieved_at)) {
        return false;
    }
    return parser.offset == parser.length;
}
