#include "dataplane.h"

#include <stddef.h>
#include <stdint.h>

int LLVMFuzzerTestOneInput(const uint8_t *data, size_t size) {
  static const char *const kinds[] = {
      "batch",          "delta",        "economic-event", "frame",
      "mapping-claim",  "materialization", "ontology-module", "packet",
      "query",          "query-result", "snapshot",       "subscription",
      "universe"};
  for (size_t i = 0U; i < 13U; i++)
    (void)tw_dp_verify(kinds[i], data, size);
  return 0;
}
