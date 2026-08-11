#include "dataplane.h"

#include <limits.h>
#include <string.h>

typedef struct {
  const uint8_t *data;
  size_t length;
  size_t offset;
} cursor;

typedef struct {
  const uint8_t *data;
  size_t length;
} slice;

typedef struct {
  slice subject;
  slice predicate;
  slice origin;
  slice observed;
  uint8_t packet[32U];
} row_key;

static bool take(cursor *c, size_t count, const uint8_t **out) {
  if (count > c->length - c->offset)
    return false;
  *out = c->data + c->offset;
  c->offset += count;
  return true;
}

static bool head(cursor *c, uint8_t *major, uint64_t *value) {
  const uint8_t *p = NULL;
  if (!take(c, 1U, &p))
    return false;
  *major = (uint8_t)(p[0] >> 5U);
  const uint8_t ai = (uint8_t)(p[0] & 31U);
  if (ai < 24U) {
    *value = ai;
    return true;
  }
  size_t count = 0U;
  if (ai == 24U)
    count = 1U;
  else if (ai == 25U)
    count = 2U;
  else if (ai == 26U)
    count = 4U;
  else if (ai == 27U)
    count = 8U;
  else
    return false;
  if (!take(c, count, &p))
    return false;
  uint64_t result = 0U;
  for (size_t i = 0U; i < count; i++)
    result = (result << 8U) | p[i];
  if ((count == 1U && result < 24U) || (count == 2U && result <= UINT8_MAX) ||
      (count == 4U && result <= UINT16_MAX) ||
      (count == 8U && result <= UINT32_MAX))
    return false;
  *value = result;
  return true;
}

static bool array(cursor *c, uint64_t minimum, uint64_t maximum,
                  uint64_t *count) {
  uint8_t major = 0U;
  uint64_t value = 0U;
  if (!head(c, &major, &value) || major != 4U || value < minimum ||
      value > maximum)
    return false;
  *count = value;
  return true;
}

static bool exact_array(cursor *c, uint64_t expected) {
  uint64_t count = 0U;
  return array(c, expected, expected, &count);
}

static bool uint_value(cursor *c, uint64_t maximum, uint64_t *value) {
  uint8_t major = 0U;
  uint64_t parsed = 0U;
  if (!head(c, &major, &parsed) || major != 0U || parsed > maximum)
    return false;
  *value = parsed;
  return true;
}

static bool is_valid_utf8(const uint8_t *s, size_t n) {
  size_t i = 0U;
  while (i < n) {
    const uint8_t b = s[i++];
    if (b < 0x80U) {
      if (b == 0U)
        return false;
      continue;
    }
    uint32_t cp = 0U;
    size_t continuation = 0U;
    if (b >= 0xc2U && b <= 0xdfU) {
      cp = (uint32_t)(b & 0x1fU);
      continuation = 1U;
    } else if (b >= 0xe0U && b <= 0xefU) {
      cp = (uint32_t)(b & 0x0fU);
      continuation = 2U;
    } else if (b >= 0xf0U && b <= 0xf4U) {
      cp = (uint32_t)(b & 0x07U);
      continuation = 3U;
    } else
      return false;
    if (continuation > n - i)
      return false;
    for (size_t j = 0U; j < continuation; j++) {
      const uint8_t x = s[i++];
      if ((x & 0xc0U) != 0x80U)
        return false;
      cp = (cp << 6U) | (uint32_t)(x & 0x3fU);
    }
    if ((continuation == 1U && cp < 0x80U) ||
        (continuation == 2U && cp < 0x800U) ||
        (continuation == 3U && cp < 0x10000U) || cp > 0x10ffffU ||
        (cp >= 0xd800U && cp <= 0xdfffU))
      return false;
  }
  return true;
}

static bool text(cursor *c, size_t minimum, size_t maximum, slice *out) {
  uint8_t major = 0U;
  uint64_t length = 0U;
  const uint8_t *p = NULL;
  if (!head(c, &major, &length) || major != 3U || length < (uint64_t)minimum ||
      length > (uint64_t)maximum || length > SIZE_MAX)
    return false;
  if (!take(c, (size_t)length, &p) || !is_valid_utf8(p, (size_t)length))
    return false;
  out->data = p;
  out->length = (size_t)length;
  return true;
}

static bool slice_eq(slice value, const char *literal) {
  const size_t n = strlen(literal);
  return value.length == n && memcmp(value.data, literal, n) == 0;
}

static int slice_cmp(slice a, slice b) {
  const size_t n = a.length < b.length ? a.length : b.length;
  const int cmp = memcmp(a.data, b.data, n);
  if (cmp != 0)
    return cmp;
  return (a.length > b.length) - (a.length < b.length);
}

static bool text_literal(cursor *c, const char *literal) {
  slice value = {0};
  return text(c, strlen(literal), strlen(literal), &value) &&
         slice_eq(value, literal);
}

static bool one_of(slice value, const char *const *allowed, size_t count) {
  for (size_t i = 0U; i < count; i++)
    if (slice_eq(value, allowed[i]))
      return true;
  return false;
}

static bool enum_text(cursor *c, const char *const *allowed, size_t count,
                      slice *out) {
  slice value = {0};
  if (!text(c, 1U, 512U, &value) || !one_of(value, allowed, count))
    return false;
  if (out != NULL)
    *out = value;
  return true;
}

static bool digest(cursor *c, uint8_t out[32U]) {
  uint8_t major = 0U;
  uint64_t length = 0U;
  const uint8_t *p = NULL;
  if (!head(c, &major, &length) || major != 2U || length != 32U ||
      !take(c, 32U, &p))
    return false;
  if (out != NULL)
    memcpy(out, p, 32U);
  return true;
}

static bool optional_digest(cursor *c, bool *present, uint8_t out[32U]) {
  if (c->offset >= c->length)
    return false;
  if (c->data[c->offset] == 0xf6U) {
    c->offset++;
    *present = false;
    if (out != NULL)
      memset(out, 0, 32U);
    return true;
  }
  *present = true;
  return digest(c, out);
}

static bool optional_text(cursor *c, size_t maximum, bool *present,
                          slice *out) {
  if (c->offset >= c->length)
    return false;
  if (c->data[c->offset] == 0xf6U) {
    c->offset++;
    *present = false;
    out->data = NULL;
    out->length = 0U;
    return true;
  }
  *present = true;
  return text(c, 1U, maximum, out);
}

static bool boolean_uint(cursor *c, bool *value) {
  uint64_t parsed = 0U;
  if (!uint_value(c, 1U, &parsed))
    return false;
  *value = parsed == 1U;
  return true;
}

static bool empty_extensions(cursor *c) { return exact_array(c, 0U); }

static bool calendar_date(slice value) {
  if (value.length != 10U || value.data[4] != '-' || value.data[7] != '-')
    return false;
  for (size_t i = 0U; i < 10U; i++) {
    if (i == 4U || i == 7U)
      continue;
    if (value.data[i] < '0' || value.data[i] > '9')
      return false;
  }
  const unsigned year = (unsigned)(value.data[0] - '0') * 1000U +
                        (unsigned)(value.data[1] - '0') * 100U +
                        (unsigned)(value.data[2] - '0') * 10U +
                        (unsigned)(value.data[3] - '0');
  const unsigned month =
      (unsigned)(value.data[5] - '0') * 10U + (unsigned)(value.data[6] - '0');
  const unsigned day =
      (unsigned)(value.data[8] - '0') * 10U + (unsigned)(value.data[9] - '0');
  if (month < 1U || month > 12U)
    return false;
  static const unsigned days[] = {31U, 28U, 31U, 30U, 31U, 30U,
                                  31U, 31U, 30U, 31U, 30U, 31U};
  unsigned maximum = days[month - 1U];
  const bool leap =
      (year % 4U == 0U && year % 100U != 0U) || (year % 400U == 0U);
  if (month == 2U && leap)
    maximum = 29U;
  return day >= 1U && day <= maximum;
}

static bool timestamp(slice value) {
  if (value.length != 20U)
    return false;
  const uint8_t *s = value.data;
  for (size_t i = 0U; i < 20U; i++) {
    const bool digit = (i <= 3U) || (i >= 5U && i <= 6U) ||
                       (i >= 8U && i <= 9U) || (i >= 11U && i <= 12U) ||
                       (i >= 14U && i <= 15U) || (i >= 17U && i <= 18U);
    if (digit && (s[i] < '0' || s[i] > '9'))
      return false;
  }
  if (s[4] != '-' || s[7] != '-' || s[10] != 'T' || s[13] != ':' ||
      s[16] != ':' || s[19] != 'Z')
    return false;
  const unsigned hour = (unsigned)(s[11] - '0') * 10U + (unsigned)(s[12] - '0');
  const unsigned minute =
      (unsigned)(s[14] - '0') * 10U + (unsigned)(s[15] - '0');
  const unsigned second =
      (unsigned)(s[17] - '0') * 10U + (unsigned)(s[18] - '0');
  const slice date = {s, 10U};
  return calendar_date(date) && hour <= 23U && minute <= 59U && second <= 59U;
}

static bool timestamp_text(cursor *c, slice *out) {
  return text(c, 20U, 20U, out) && timestamp(*out);
}
static bool optional_timestamp(cursor *c, bool *present, slice *out) {
  return optional_text(c, 20U, present, out) && (!*present || timestamp(*out));
}

static bool currency(slice value) {
  if (value.length != 3U)
    return false;
  for (size_t i = 0U; i < 3U; i++)
    if (value.data[i] < 'A' || value.data[i] > 'Z')
      return false;
  return true;
}

static bool canonical_integer(slice value) {
  size_t i = 0U;
  if (value.length == 0U)
    return false;
  if (value.data[0] == '-') {
    i = 1U;
    if (i == value.length)
      return false;
  }
  if (value.data[i] == '0' && i + 1U < value.length)
    return false;
  for (; i < value.length; i++)
    if (value.data[i] < '0' || value.data[i] > '9')
      return false;
  return true;
}
static bool canonical_decimal(slice value) {
  size_t dot = value.length;
  for (size_t i = 0U; i < value.length; i++)
    if (value.data[i] == '.') {
      if (dot != value.length)
        return false;
      dot = i;
    }
  if (dot == value.length || dot + 1U == value.length)
    return false;
  slice whole = {value.data, dot};
  slice fraction = {value.data + dot + 1U, value.length - dot - 1U};
  if (!canonical_integer(whole))
    return false;
  for (size_t i = 0U; i < fraction.length; i++)
    if (fraction.data[i] < '0' || fraction.data[i] > '9')
      return false;
  return true;
}

static bool absolute_uri(slice value) {
  if (value.length < 3U || !((value.data[0] >= 'A' && value.data[0] <= 'Z') ||
                             (value.data[0] >= 'a' && value.data[0] <= 'z')))
    return false;
  size_t separator = value.length;
  for (size_t i = 0U; i < value.length; i++) {
    const uint8_t b = value.data[i];
    if (b <= 0x20U || b == 0x7fU)
      return false;
    if (b == ':' && separator == value.length)
      separator = i;
  }
  if (separator < 1U || separator + 1U >= value.length)
    return false;
  for (size_t i = 1U; i < separator; i++) {
    const uint8_t b = value.data[i];
    if (!((b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') ||
          (b >= '0' && b <= '9') || b == '+' || b == '-' || b == '.'))
      return false;
  }
  return true;
}

static bool duration_digits(slice value, size_t *offset) {
  const size_t start = *offset;
  while (*offset < value.length && value.data[*offset] >= '0' &&
         value.data[*offset] <= '9')
    (*offset)++;
  return *offset > start;
}

static bool duration(slice value) {
  if (value.length < 2U || value.data[0] != 'P')
    return false;
  size_t offset = 1U;
  bool seen = false;
  const size_t day_start = offset;
  if (duration_digits(value, &offset)) {
    if (offset < value.length && value.data[offset] == 'D') {
      offset++;
      seen = true;
    } else {
      offset = day_start;
    }
  }
  if (offset < value.length && value.data[offset] == 'T') {
    offset++;
    bool time_seen = false;
    static const uint8_t units[] = {'H', 'M', 'S'};
    for (size_t u = 0U; u < 3U; u++) {
      const size_t start = offset;
      if (!duration_digits(value, &offset)) {
        offset = start;
        continue;
      }
      if (units[u] == 'S' && offset < value.length &&
          value.data[offset] == '.') {
        offset++;
        if (!duration_digits(value, &offset))
          return false;
      }
      if (offset < value.length && value.data[offset] == units[u]) {
        offset++;
        time_seen = true;
        seen = true;
      } else {
        offset = start;
      }
    }
    if (!time_seen)
      return false;
  }
  return seen && offset == value.length;
}

static bool language_tag(slice value) {
  if (value.length < 2U || value.length > 63U)
    return false;
  for (size_t i = 0U; i < value.length; i++) {
    const uint8_t b = value.data[i];
    if (!((b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') ||
          (b >= '0' && b <= '9') || b == '-'))
      return false;
  }
  return true;
}

static bool text_set(cursor *c, uint64_t minimum, uint64_t maximum,
                     const char *const *allowed, size_t allowed_count) {
  uint64_t count = 0U;
  if (!array(c, minimum, maximum, &count))
    return false;
  slice previous = {0};
  for (uint64_t i = 0U; i < count; i++) {
    slice current = {0};
    if (!text(c, 1U, 512U, &current) ||
        (i > 0U && slice_cmp(previous, current) >= 0))
      return false;
    if (allowed != NULL && !one_of(current, allowed, allowed_count))
      return false;
    previous = current;
  }
  return true;
}

static bool digest_set(cursor *c, uint64_t minimum, uint64_t maximum) {
  uint64_t count = 0U;
  if (!array(c, minimum, maximum, &count))
    return false;
  uint8_t previous[32U] = {0};
  for (uint64_t i = 0U; i < count; i++) {
    uint8_t current[32U];
    if (!digest(c, current) || (i > 0U && memcmp(previous, current, 32U) >= 0))
      return false;
    memcpy(previous, current, 32U);
  }
  return true;
}

static bool typed_value(cursor *c) {
  static const char *const types[] = {"boolean",  "integer", "decimal",
                                      "text",     "date",    "datetime",
                                      "duration", "uri",     "identifier"};
  if (!exact_array(c, 4U))
    return false;
  slice type = {0}, lexical = {0};
  if (!enum_text(c, types, 9U, &type) || !text(c, 1U, 262144U, &lexical))
    return false;
  bool present = false;
  slice unit = {0};
  if (!optional_text(c, 512U, &present, &unit))
    return false;
  slice money = {0};
  if (!optional_text(c, 3U, &present, &money) || (present && !currency(money)))
    return false;
  if (slice_eq(type, "boolean") &&
      !(slice_eq(lexical, "true") || slice_eq(lexical, "false")))
    return false;
  if (slice_eq(type, "integer") && !canonical_integer(lexical))
    return false;
  if (slice_eq(type, "decimal") && !canonical_decimal(lexical))
    return false;
  if (slice_eq(type, "date") && !calendar_date(lexical))
    return false;
  if (slice_eq(type, "datetime") && !timestamp(lexical))
    return false;
  if (slice_eq(type, "duration") && !duration(lexical))
    return false;
  if (slice_eq(type, "uri") && !absolute_uri(lexical))
    return false;
  if (slice_eq(type, "identifier") && lexical.length > 512U)
    return false;
  return true;
}

static bool optional_typed(cursor *c, bool *present) {
  if (c->offset >= c->length)
    return false;
  if (c->data[c->offset] == 0xf6U) {
    c->offset++;
    *present = false;
    return true;
  }
  *present = true;
  return typed_value(c);
}

static bool packet(cursor *c) {
  static const char *const kinds[] = {
      "claim",        "state", "capability",  "offer",
      "relationship", "event", "measurement", "document"};
  static const char *const statuses[] = {
      "resolved",       "unknown",  "not_observed",    "not_provided",
      "not_applicable", "withheld", "redacted",        "unresolved",
      "contradictory",  "invalid",  "confirmed_absent"};
  static const char *const lanes[] = {"observed_native", "provisional_semantic",
                                      "attested_semantic"};
  static const char *const extraction[] = {"deterministic",
                                           "publisher_attested"};
  static const char *const mappings[] = {"none", "candidate", "reviewed",
                                         "disputed", "revoked"};
  static const char *const freshness[] = {"current", "stale", "unknown"};
  static const char *const lifecycle[] = {"current", "superseded", "withdrawn",
                                          "stale",   "retracted",  "invalid"};
  static const char *const retention[] = {
      "public_transient", "public_versioned", "public_archival"};
  if (!exact_array(c, 14U) || !text_literal(c, "tw.semantic-packet/0.1") ||
      !enum_text(c, kinds, 8U, NULL) || !exact_array(c, 2U))
    return false;
  slice ignored = {0};
  if (!text(c, 1U, 512U, &ignored) || !text_set(c, 0U, 32U, NULL, 0U) ||
      !exact_array(c, 2U) || !text(c, 1U, 512U, &ignored))
    return false;
  bool semantic = false;
  slice semantic_value = {0};
  if (!optional_text(c, 512U, &semantic, &semantic_value) ||
      !exact_array(c, 5U))
    return false;
  slice status = {0}, native = {0};
  if (!enum_text(c, statuses, 11U, &status) || !text(c, 0U, 262144U, &native))
    return false;
  bool present = false;
  slice optional = {0};
  if (!optional_text(c, 255U, &present, &optional))
    return false;
  bool object_language = false;
  slice object_language_value = {0};
  if (!optional_text(c, 63U, &object_language, &object_language_value) ||
      (object_language && !language_tag(object_language_value)))
    return false;
  bool typed = false;
  if (!optional_typed(c, &typed))
    return false;
  if (!slice_eq(status, "resolved") && (native.length != 0U || typed))
    return false;
  if (!exact_array(c, 4U))
    return false;
  uint64_t dimensions = 0U;
  if (!array(c, 0U, 32U, &dimensions))
    return false;
  slice prior = {0};
  for (uint64_t i = 0U; i < dimensions; i++) {
    slice key = {0};
    if (!exact_array(c, 2U) || !text(c, 1U, 512U, &key) ||
        (i > 0U && slice_cmp(prior, key) >= 0) || !typed_value(c))
      return false;
    prior = key;
  }
  if (!optional_text(c, 512U, &present, &optional))
    return false;
  bool context_language = false;
  slice context_language_value = {0};
  if (!optional_text(c, 63U, &context_language, &context_language_value) ||
      (context_language && !language_tag(context_language_value)) ||
      !optional_text(c, 512U, &present, &optional))
    return false;
  if (!exact_array(c, 5U))
    return false;
  slice observed = {0};
  if (!timestamp_text(c, &observed))
    return false;
  for (size_t i = 0U; i < 4U; i++)
    if (!optional_timestamp(c, &present, &optional))
      return false;
  if (!exact_array(c, 4U) || !text(c, 1U, 512U, &ignored) || !digest(c, NULL) ||
      !text(c, 1U, 16384U, &ignored) ||
      !optional_text(c, 512U, &present, &optional))
    return false;
  if (!exact_array(c, 9U) || !digest(c, NULL))
    return false;
  bool digest_present = false;
  if (!optional_digest(c, &digest_present, NULL) || !digest(c, NULL) ||
      !digest(c, NULL))
    return false;
  uint64_t transforms = 0U;
  if (!array(c, 0U, 32U, &transforms))
    return false;
  slice transform_views[32U];
  for (uint64_t i = 0U; i < transforms; i++) {
    slice v = {0};
    if (!text(c, 1U, 512U, &v))
      return false;
    for (uint64_t j = 0U; j < i; j++)
      if (slice_cmp(transform_views[j], v) == 0)
        return false;
    transform_views[i] = v;
  }
  const size_t mapping_start = c->offset;
  uint64_t mapping_count = 0U;
  if (!array(c, 0U, 32U, &mapping_count))
    return false;
  prior = (slice){0};
  for (uint64_t i = 0U; i < mapping_count; i++) {
    slice v = {0};
    if (!text(c, 1U, 512U, &v) || (i > 0U && slice_cmp(prior, v) >= 0))
      return false;
    prior = v;
  }
  (void)mapping_start;
  bool closure = false;
  if (!optional_digest(c, &closure, NULL) || !digest(c, NULL) ||
      !text(c, 1U, 512U, &ignored) || !exact_array(c, 6U))
    return false;
  slice lane = {0}, mapping = {0};
  if (!enum_text(c, lanes, 3U, &lane) || !enum_text(c, extraction, 2U, NULL) ||
      !enum_text(c, mappings, 5U, &mapping))
    return false;
  bool confidence = false;
  uint64_t confidence_value = 0U;
  if (c->offset < c->length && c->data[c->offset] == 0xf6U) {
    c->offset++;
  } else {
    confidence = true;
    if (!uint_value(c, 1000000U, &confidence_value))
      return false;
  }
  (void)confidence_value;
  if (!text(c, 1U, 512U, &ignored) || !enum_text(c, freshness, 3U, NULL))
    return false;
  if (slice_eq(lane, "observed_native") &&
      (semantic || mapping_count > 0U || closure ||
       !slice_eq(mapping, "none") || confidence))
    return false;
  if (slice_eq(lane, "provisional_semantic") &&
      (!semantic || mapping_count == 0U || !closure ||
       !slice_eq(mapping, "candidate")))
    return false;
  if (slice_eq(lane, "attested_semantic") &&
      (!semantic || mapping_count == 0U || !closure ||
       !slice_eq(mapping, "reviewed") || confidence))
    return false;
  if (confidence && !slice_eq(mapping, "candidate"))
    return false;
  if (!exact_array(c, 2U))
    return false;
  slice state = {0};
  if (!enum_text(c, lifecycle, 6U, &state) ||
      !optional_digest(c, &digest_present, NULL))
    return false;
  if (slice_eq(state, "superseded") && !digest_present)
    return false;
  if (slice_eq(state, "current") && digest_present)
    return false;
  if (!enum_text(c, retention, 3U, NULL) || !text_literal(c, "public") ||
      !empty_extensions(c))
    return false;
  return c->offset == c->length;
}

static bool digest_entries(cursor *c, uint64_t maximum, uint64_t minimum) {
  uint64_t count = 0U;
  if (!array(c, minimum, maximum, &count))
    return false;
  uint8_t previous[32U] = {0};
  for (uint64_t i = 0U; i < count; i++) {
    uint8_t current[32U];
    uint64_t size = 0U;
    if (!exact_array(c, 2U) || !digest(c, current) ||
        !uint_value(c, 4194304U, &size) || size == 0U ||
        (i > 0U && memcmp(previous, current, 32U) >= 0))
      return false;
    memcpy(previous, current, 32U);
  }
  return true;
}

static bool batch(cursor *c) {
  if (!exact_array(c, 14U) ||
      !text_literal(c, "tw.packet-batch-manifest/0.1")) {
    return false;
  }
  slice v = {0}, start = {0}, end = {0};
  if (!text(c, 1U, 512U, &v) || !digest(c, NULL) || !digest(c, NULL) ||
      !timestamp_text(c, &start) || !timestamp_text(c, &end) ||
      slice_cmp(start, end) > 0)
    return false;
  bool present = false;
  if (!optional_digest(c, &present, NULL) || !digest_set(c, 1U, 1024U) ||
      !digest_entries(c, 32768U, 0U) || !digest_entries(c, 32768U, 0U) ||
      !digest(c, NULL) || !digest(c, NULL))
    return false;
  uint64_t count = 0U;
  if (!array(c, 1U, 64U, &count))
    return false;
  slice prior = {0};
  for (uint64_t i = 0U; i < count; i++) {
    slice name = {0};
    uint64_t size = 0U;
    if (!exact_array(c, 3U) || !text(c, 1U, 255U, &name) ||
        (i > 0U && slice_cmp(prior, name) >= 0) || !digest(c, NULL) ||
        !uint_value(c, 4194304U, &size) || size == 0U)
      return false;
    prior = name;
  }
  return empty_extensions(c) && c->offset == c->length;
}

static bool delta(cursor *c) {
  static const char *const classes[] = {"origin", "semantic", "canon"};
  static const char *const origin[] = {"added", "modified", "withdrawn",
                                       "restored", "source_retracted"};
  static const char *const semantic[] = {"mapped",     "remapped", "narrowed",
                                         "broadened",  "disputed", "attested",
                                         "de_attested"};
  static const char *const canon[] = {"module_added", "module_superseded",
                                      "mapping_superseded", "closure_changed"};
  if (!exact_array(c, 14U) || !text_literal(c, "tw.semantic-delta/0.1"))
    return false;
  slice class = {0}, kind = {0};
  if (!enum_text(c, classes, 3U, &class) || !text(c, 1U, 512U, &kind))
    return false;
  const char *const *allowed = origin;
  size_t count = 5U;
  if (slice_eq(class, "semantic")) {
    allowed = semantic;
    count = 7U;
  } else if (slice_eq(class, "canon")) {
    allowed = canon;
    count = 4U;
  }
  if (!one_of(kind, allowed, count) || !digest(c, NULL))
    return false;
  bool bp = false, ap = false, be = false, ae = false;
  uint8_t before_e[32U], after_e[32U], before_p[32U], after_p[32U];
  if (!optional_digest(c, &bp, before_p) || !optional_digest(c, &ap, after_p) ||
      !optional_digest(c, &be, before_e) || !optional_digest(c, &ae, after_e))
    return false;
  slice v = {0};
  if (!text(c, 1U, 512U, &v) || !timestamp_text(c, &v) || !digest(c, NULL) ||
      !text(c, 1U, 512U, &v) || !text(c, 1U, 512U, &v) ||
      !empty_extensions(c) || c->offset != c->length)
    return false;
  if (slice_eq(class, "origin")) {
    if (slice_eq(kind, "added") && (bp || be || !ap || !ae))
      return false;
    if ((slice_eq(kind, "modified") || slice_eq(kind, "restored")) &&
        (!bp || !ap || !be || !ae || memcmp(before_e, after_e, 32U) == 0))
      return false;
    if ((slice_eq(kind, "withdrawn") || slice_eq(kind, "source_retracted")) &&
        (!bp || ap || !be))
      return false;
  } else {
    if (!bp || !ap || !be || !ae || memcmp(before_e, after_e, 32U) != 0 ||
        memcmp(before_p, after_p, 32U) == 0)
      return false;
  }
  return true;
}

static bool query_dimension_values(cursor *c) {
  uint64_t count = 0U;
  if (!array(c, 1U, 64U, &count))
    return false;
  slice prior = {0};
  for (uint64_t i = 0U; i < count; i++) {
    const size_t start = c->offset;
    if (!typed_value(c))
      return false;
    slice current = {c->data + start, c->offset - start};
    if (i > 0U && slice_cmp(prior, current) >= 0)
      return false;
    prior = current;
  }
  return true;
}

static bool query(cursor *c) {
  static const char *const relations[] = {"eq", "in", "lt", "lte", "gt", "gte"};
  static const char *const time_modes[] = {"current", "as_of", "between",
                                           "history"};
  static const char *const edges[] = {"candidate", "reviewed", "disputed"};
  static const char *const lanes[] = {"observed_native", "provisional_semantic",
                                      "attested_semantic"};
  static const char *const mappings[] = {"none", "candidate", "reviewed",
                                         "disputed"};
  static const char *const stale[] = {"exclude", "return_explicit_stale",
                                      "request_refresh"};
  static const char *const conflicts[] = {
      "preserve_sources", "group_equivalent", "reject_conflict"};
  static const char *const proof[] = {"packet", "field", "bundle"};
  static const char *const prefs[] = {
      "most_authoritative", "freshest",      "fastest",
      "least_expensive",    "highest_proof", "balanced"};
  if (!exact_array(c, 16U) || !text_literal(c, "tw.semantic-query/0.1") ||
      !text_set(c, 1U, 32U, NULL, 0U) || !exact_array(c, 2U))
    return false;
  bool concept = false;
  slice optional = {0};
  if (!optional_text(c, 512U, &concept, &optional))
    return false;
  const size_t ids_start = c->offset;
  uint64_t id_count = 0U;
  if (!array(c, 0U, 256U, &id_count))
    return false;
  slice prior = {0};
  for (uint64_t i = 0U; i < id_count; i++) {
    slice v = {0};
    if (!text(c, 1U, 512U, &v) || (i > 0U && slice_cmp(prior, v) >= 0))
      return false;
    prior = v;
  }
  (void)ids_start;
  if (!concept && id_count == 0U)
    return false;
  uint64_t dimensions = 0U;
  if (!array(c, 0U, 32U, &dimensions))
    return false;
  prior = (slice){0};
  for (uint64_t i = 0U; i < dimensions; i++) {
    slice key = {0};
    if (!exact_array(c, 3U) || !text(c, 1U, 512U, &key) ||
        (i > 0U && slice_cmp(prior, key) >= 0) ||
        !enum_text(c, relations, 6U, NULL) || !query_dimension_values(c))
      return false;
    prior = key;
  }
  if (!exact_array(c, 3U))
    return false;
  slice mode = {0}, from = {0}, until = {0};
  bool fp = false, up = false;
  if (!enum_text(c, time_modes, 4U, &mode) ||
      !optional_timestamp(c, &fp, &from) || !optional_timestamp(c, &up, &until))
    return false;
  if (slice_eq(mode, "current") && (fp || up))
    return false;
  if (slice_eq(mode, "as_of") && (fp || !up))
    return false;
  if (slice_eq(mode, "between") && (!fp || !up || slice_cmp(from, until) > 0))
    return false;
  if (slice_eq(mode, "history") && fp && up && slice_cmp(from, until) > 0)
    return false;
  uint64_t value = 0U;
  if (!exact_array(c, 3U) || !uint_value(c, 16U, &value) ||
      !uint_value(c, 16000000U, &value) || !text_set(c, 1U, 3U, edges, 3U) ||
      !exact_array(c, 3U))
    return false;
  uint64_t origin_count = 0U;
  if (!array(c, 0U, 256U, &origin_count))
    return false;
  prior = (slice){0};
  for (uint64_t i = 0U; i < origin_count; i++) {
    slice v = {0};
    if (!text(c, 1U, 512U, &v) || (i > 0U && slice_cmp(prior, v) >= 0))
      return false;
    prior = v;
  }
  uint64_t minimum_origins = 0U;
  if (!uint_value(c, 32U, &minimum_origins) || minimum_origins == 0U ||
      (origin_count > 0U && minimum_origins > origin_count) ||
      !text_set(c, 0U, 32U, NULL, 0U))
    return false;
  if (!exact_array(c, 2U) || !text_set(c, 1U, 3U, lanes, 3U) ||
      !text_set(c, 1U, 4U, mappings, 4U) || !exact_array(c, 2U))
    return false;
  bool max_age = false;
  if (c->offset < c->length && c->data[c->offset] == 0xf6U)
    c->offset++;
  else {
    max_age = true;
    if (!uint_value(c, 315576000U, &value))
      return false;
  }
  (void)max_age;
  slice stale_mode = {0};
  if (!enum_text(c, stale, 3U, &stale_mode) || !exact_array(c, 3U))
    return false;
  bool price = false, currency_present = false;
  slice price_value = {0}, currency_value = {0};
  if (!optional_text(c, 128U, &price, &price_value) ||
      !optional_text(c, 3U, &currency_present, &currency_value))
    return false;
  if (price &&
      !(canonical_integer(price_value) || canonical_decimal(price_value)))
    return false;
  if (price != currency_present ||
      (currency_present && !currency(currency_value)) ||
      !text_set(c, 0U, 16U, NULL, 0U) || !exact_array(c, 1U) ||
      !enum_text(c, conflicts, 3U, NULL) || !exact_array(c, 4U))
    return false;
  bool materialized = false, live = false;
  if (!boolean_uint(c, &materialized) || !boolean_uint(c, &live))
    return false;
  uint64_t live_count = 0U, deadline = 0U;
  if (!uint_value(c, 8U, &live_count) || !uint_value(c, 30000U, &deadline) ||
      deadline == 0U || (!live && live_count != 0U) ||
      (!materialized && !live) ||
      (slice_eq(stale_mode, "request_refresh") && !live))
    return false;
  if (!exact_array(c, 3U) || !enum_text(c, proof, 3U, NULL))
    return false;
  bool include = false, native = false;
  if (!boolean_uint(c, &include) || !boolean_uint(c, &native) || !native ||
      !enum_text(c, prefs, 6U, NULL) || !exact_array(c, 3U))
    return false;
  uint64_t results = 0U, packets = 0U, proof_bytes = 0U;
  if (!uint_value(c, 1000U, &results) || results == 0U ||
      !uint_value(c, 10000U, &packets) || packets == 0U ||
      !uint_value(c, 16777216U, &proof_bytes) || proof_bytes < 1024U ||
      !empty_extensions(c))
    return false;
  return c->offset == c->length;
}

static bool subscription(cursor *c) {
  static const char *const classes[] = {"origin", "semantic", "canon"};
  static const char *const kinds[] = {
      "added",          "modified",          "withdrawn",
      "restored",       "source_retracted",  "mapped",
      "remapped",       "narrowed",          "broadened",
      "disputed",       "attested",          "de_attested",
      "module_added",   "module_superseded", "mapping_superseded",
      "closure_changed"};
  static const char *const delivery[] = {"sse", "poll"};
  static const char *const proof[] = {"packet", "bundle"};
  if (!exact_array(c, 10U) ||
      !text_literal(c, "tw.semantic-subscription/0.1") || !digest(c, NULL) ||
      !text_set(c, 1U, 3U, classes, 3U) || !text_set(c, 1U, 16U, kinds, 16U) ||
      !enum_text(c, delivery, 2U, NULL))
    return false;
  uint64_t v = 0U;
  if (!uint_value(c, UINT64_MAX, &v) || !uint_value(c, 600U, &v) || v == 0U ||
      !enum_text(c, proof, 2U, NULL))
    return false;
  bool present = false;
  slice expires = {0};
  return optional_timestamp(c, &present, &expires) && empty_extensions(c) &&
         c->offset == c->length;
}

static int row_compare(row_key a, row_key b) {
  int cmp = slice_cmp(a.subject, b.subject);
  if (cmp != 0)
    return cmp;
  cmp = slice_cmp(a.predicate, b.predicate);
  if (cmp != 0)
    return cmp;
  cmp = slice_cmp(a.origin, b.origin);
  if (cmp != 0)
    return cmp;
  cmp = slice_cmp(a.observed, b.observed);
  if (cmp != 0)
    return cmp;
  return memcmp(a.packet, b.packet, 32U);
}
static bool result_row(cursor *c, row_key *out) {
  static const char *const statuses[] = {
      "resolved",       "unknown",  "not_observed",    "not_provided",
      "not_applicable", "withheld", "redacted",        "unresolved",
      "contradictory",  "invalid",  "confirmed_absent"};
  static const char *const lanes[] = {"observed_native", "provisional_semantic",
                                      "attested_semantic"};
  if (!exact_array(c, 13U) || !text(c, 1U, 512U, &out->subject) ||
      !text(c, 1U, 512U, &out->predicate))
    return false;
  slice status = {0}, ignored = {0};
  if (!enum_text(c, statuses, 11U, &status) || !text(c, 1U, 512U, &ignored) ||
      !text(c, 1U, 16384U, &ignored))
    return false;
  slice lexical = {0};
  if (!text(c, 0U, 262144U, &lexical))
    return false;
  bool semantic = false, typed = false;
  if (!optional_text(c, 512U, &semantic, &ignored) ||
      !optional_typed(c, &typed) || !text(c, 1U, 512U, &out->origin) ||
      !digest(c, out->packet) || !digest(c, NULL))
    return false;
  slice lane = {0};
  if (!enum_text(c, lanes, 3U, &lane) || !timestamp_text(c, &out->observed))
    return false;
  if (!slice_eq(status, "resolved") && (lexical.length != 0U || typed))
    return false;
  if (slice_eq(lane, "observed_native") && semantic)
    return false;
  if (!slice_eq(lane, "observed_native") && !semantic)
    return false;
  return true;
}

static bool query_result(cursor *c) {
  static const char *const prefs[] = {
      "most_authoritative", "freshest",      "fastest",
      "least_expensive",    "highest_proof", "balanced"};
  static const char *const statuses[] = {"resolved", "partial", "unresolved"};
  static const char *const conflict_kinds[] = {"value", "time", "identity",
                                               "mapping", "authority"};
  static const char *const resolutions[] = {
      "preserved", "caller_policy_selected", "unresolved"};
  if (!exact_array(c, 12U) ||
      !text_literal(c, "tw.semantic-query-result/0.1") || !digest(c, NULL) ||
      !digest(c, NULL) || !enum_text(c, prefs, 6U, NULL))
    return false;
  uint64_t sequence = 0U;
  if (!uint_value(c, UINT64_MAX, &sequence))
    return false;
  slice status = {0};
  if (!enum_text(c, statuses, 3U, &status))
    return false;
  uint64_t rows = 0U;
  if (!array(c, 0U, 1000U, &rows))
    return false;
  row_key prior = {0};
  for (uint64_t i = 0U; i < rows; i++) {
    row_key current = {0};
    if (!result_row(c, &current) ||
        (i > 0U && row_compare(prior, current) >= 0))
      return false;
    prior = current;
  }
  if (slice_eq(status, "resolved") && rows == 0U)
    return false;
  if (slice_eq(status, "unresolved") && rows != 0U)
    return false;
  uint64_t conflicts = 0U;
  if (!array(c, 0U, 256U, &conflicts))
    return false;
  uint8_t prior_key[32U] = {0};
  for (uint64_t i = 0U; i < conflicts; i++) {
    uint8_t key[32U];
    if (!exact_array(c, 4U) || !digest(c, key) ||
        (i > 0U && memcmp(prior_key, key, 32U) >= 0) ||
        !enum_text(c, conflict_kinds, 5U, NULL) || !digest_set(c, 2U, 32U) ||
        !enum_text(c, resolutions, 3U, NULL))
      return false;
    memcpy(prior_key, key, 32U);
  }
  if (!digest_entries(c, 10000U, 1U) || !digest(c, NULL)) {
    return false;
  }
  slice generated = {0};
  return timestamp_text(c, &generated) && empty_extensions(c) &&
         c->offset == c->length;
}

static bool materialization(cursor *c) {
  if (!exact_array(c, 10U) ||
      !text_literal(c, "tw.materialization-manifest/0.1"))
    return false;
  slice v = {0};
  if (!text(c, 1U, 512U, &v) || !digest(c, NULL) || !text(c, 1U, 512U, &v))
    return false;
  uint64_t n = 0U;
  if (!uint_value(c, UINT64_MAX, &n) || !digest_set(c, 0U, 100000U) ||
      !digest(c, NULL) || !uint_value(c, 10000000U, &n) ||
      !timestamp_text(c, &v) || !empty_extensions(c))
    return false;
  return c->offset == c->length;
}

static bool money(cursor *c) {
  if (!exact_array(c, 3U))
    return false;
  slice currency_value = {0}, amount = {0}, class_value = {0};
  return text(c, 3U, 3U, &currency_value) && currency(currency_value) &&
         text(c, 1U, 128U, &amount) &&
         (canonical_integer(amount) || canonical_decimal(amount)) &&
         text(c, 1U, 512U, &class_value);
}
static bool economic(cursor *c) {
  if (!exact_array(c, 13U) || !text_literal(c, "tw.economic-event/0.1"))
    return false;
  slice v = {0};
  if (!text(c, 1U, 512U, &v) || !timestamp_text(c, &v))
    return false;
  bool present = false;
  if (!optional_text(c, 512U, &present, &v) || !text(c, 1U, 512U, &v) ||
      !optional_digest(c, &present, NULL) ||
      !optional_digest(c, &present, NULL) || !exact_array(c, 7U))
    return false;
  const uint64_t maximums[] = {1000000U,       1099511627776U, 4294967295U,
                               1099511627776U, 1099511627776U, 1099511627776U,
                               4294967295U};
  uint64_t value = 0U;
  for (size_t i = 0U; i < 7U; i++)
    if (!uint_value(c, maximums[i], &value))
      return false;
  if (!text(c, 1U, 512U, &v) || !optional_text(c, 512U, &present, &v) ||
      !money(c) || !money(c) || !text(c, 1U, 512U, &v) || !empty_extensions(c))
    return false;
  return c->offset == c->length;
}

static bool safe_path(slice value) {
  if (value.length == 0U || value.length > 255U || value.data[0] == '/' ||
      value.data[value.length - 1U] == '/' ||
      slice_eq(value, "manifest.cbor") || slice_eq(value, "manifest.json"))
    return false;
  if (value.length >= 9U && memcmp(value.data, "channels/", 9U) == 0)
    return false;
  size_t segment = 0U;
  for (size_t i = 0U; i <= value.length; i++) {
    if (i < value.length && (value.data[i] == '\\' || value.data[i] < 0x20U ||
                             value.data[i] == 0x7fU))
      return false;
    if (i == value.length || value.data[i] == '/') {
      const size_t n = i - segment;
      if (n == 0U || (n == 1U && value.data[segment] == '.') ||
          (n == 2U && value.data[segment] == '.' &&
           value.data[segment + 1U] == '.'))
        return false;
      for (size_t j = segment; j + 2U < i; j++) {
        if (value.data[j] == '%' && value.data[j + 1U] == '2' &&
            (value.data[j + 2U] == 'f' || value.data[j + 2U] == 'F'))
          return false;
        if (value.data[j] == '%' && value.data[j + 1U] == '5' &&
            (value.data[j + 2U] == 'c' || value.data[j + 2U] == 'C'))
          return false;
      }
      segment = i + 1U;
    }
  }
  return true;
}

static bool snapshot(cursor *c) {
  static const char *const roles[] = {
      "origin_catalog",   "concepts",          "mappings",     "packet_batch",
      "delta_batch",      "materialized_view", "search_index", "proof_index",
      "economic_summary", "build_report"};
  if (!exact_array(c, 18U) ||
      !text_literal(c, "tw.semantic-snapshot-manifest/0.1"))
    return false;
  slice v = {0};
  if (!text(c, 1U, 512U, &v) || !timestamp_text(c, &v) ||
      !text(c, 1U, 512U, &v) || !digest(c, NULL) || !text(c, 1U, 512U, &v) ||
      !digest(c, NULL) || !digest(c, NULL))
    return false;
  bool present = false;
  if (!optional_digest(c, &present, NULL) || !text_set(c, 1U, 16U, NULL, 0U))
    return false;
  uint64_t artifacts = 0U;
  if (!array(c, 1U, 16384U, &artifacts))
    return false;
  slice prior = {0};
  bool required[6U] = {false, false, false, false, false, false};
  uint64_t total = 0U;
  uint8_t build_digest[32U] = {0};
  size_t build_count = 0U;
  uint8_t materialized[32U][32U];
  size_t materialized_count = 0U;
  for (uint64_t i = 0U; i < artifacts; i++) {
    slice artifact_path = {0}, role = {0};
    uint8_t artifact_digest[32U];
    uint64_t size = 0U;
    if (!exact_array(c, 5U) || !text(c, 1U, 255U, &artifact_path) ||
        !safe_path(artifact_path) ||
        (i > 0U && slice_cmp(prior, artifact_path) >= 0) ||
        !digest(c, artifact_digest) || !uint_value(c, 4294967296U, &size) ||
        size == 0U || !text(c, 1U, 255U, &v) ||
        !enum_text(c, roles, 10U, &role))
      return false;
    if (total > 8589934592U - size)
      return false;
    total += size;
    prior = artifact_path;
    if (slice_eq(role, "origin_catalog"))
      required[0] = true;
    else if (slice_eq(role, "concepts"))
      required[1] = true;
    else if (slice_eq(role, "mappings"))
      required[2] = true;
    else if (slice_eq(role, "packet_batch"))
      required[3] = true;
    else if (slice_eq(role, "proof_index"))
      required[4] = true;
    else if (slice_eq(role, "build_report")) {
      required[5] = true;
      build_count++;
      memcpy(build_digest, artifact_digest, 32U);
    } else if (slice_eq(role, "materialized_view")) {
      if (materialized_count >= 32U)
        return false;
      memcpy(materialized[materialized_count++], artifact_digest, 32U);
    }
  }
  for (size_t i = 0U; i < 6U; i++)
    if (!required[i])
      return false;
  if (build_count != 1U)
    return false;
  uint64_t views = 0U;
  if (!array(c, 0U, 32U, &views))
    return false;
  prior = (slice){0};
  uint64_t maximum_view_sequence = 0U;
  for (uint64_t i = 0U; i < views; i++) {
    slice id = {0};
    uint8_t artifact_digest[32U];
    uint64_t row_count = 0U, through = 0U;
    if (!exact_array(c, 5U) || !text(c, 1U, 512U, &id) ||
        (i > 0U && slice_cmp(prior, id) >= 0) || !digest(c, NULL) ||
        !digest(c, artifact_digest) || !uint_value(c, 10000000U, &row_count) ||
        !uint_value(c, UINT64_MAX, &through))
      return false;
    bool found = false;
    for (size_t j = 0U; j < materialized_count; j++)
      if (memcmp(materialized[j], artifact_digest, 32U) == 0)
        found = true;
    if (!found)
      return false;
    if (through > maximum_view_sequence)
      maximum_view_sequence = through;
    prior = id;
  }
  if (!exact_array(c, 8U))
    return false;
  const uint64_t max_counts[] = {1000000U,    10000000U,   10000000U,
                                 1000000000U, 1000000000U, 32U,
                                 1000000000U, 1000000000U};
  uint64_t count_values[8U];
  for (size_t i = 0U; i < 8U; i++)
    if (!uint_value(c, max_counts[i], &count_values[i]))
      return false;
  if (count_values[5] != views)
    return false;
  uint64_t high_packet = 0U, high_delta = 0U, declared_total = 0U;
  if (!uint_value(c, UINT64_MAX, &high_packet) ||
      !uint_value(c, UINT64_MAX, &high_delta) ||
      !uint_value(c, 8589934592U, &declared_total) || declared_total != total ||
      declared_total == 0U)
    return false;
  if (maximum_view_sequence > high_packet)
    return false;
  (void)high_delta;
  for (uint64_t i = 0U; i < views; i++) {
    (void)i;
  }
  uint8_t reported_build[32U];
  if (!digest(c, reported_build) ||
      memcmp(reported_build, build_digest, 32U) != 0 || !empty_extensions(c))
    return false;
  return c->offset == c->length;
}

bool tw_dp_verify(const char *kind, const uint8_t *data, size_t length) {
  if (kind == NULL || data == NULL || length == 0U ||
      length > TW_DP_MAX_DOCUMENT_BYTES)
    return false;
  cursor c = {data, length, 0U};
  if (strcmp(kind, "packet") == 0)
    return packet(&c);
  if (strcmp(kind, "batch") == 0)
    return batch(&c);
  if (strcmp(kind, "delta") == 0)
    return delta(&c);
  if (strcmp(kind, "query") == 0)
    return query(&c);
  if (strcmp(kind, "subscription") == 0)
    return subscription(&c);
  if (strcmp(kind, "query-result") == 0)
    return query_result(&c);
  if (strcmp(kind, "materialization") == 0)
    return materialization(&c);
  if (strcmp(kind, "economic-event") == 0)
    return economic(&c);
  if (strcmp(kind, "snapshot") == 0)
    return snapshot(&c);
  return false;
}
