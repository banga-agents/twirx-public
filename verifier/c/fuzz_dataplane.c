#include "dataplane.h"

#include <stddef.h>
#include <stdint.h>

int LLVMFuzzerTestOneInput(const uint8_t *data, size_t size) {
  static const char *const kinds[] = {
      "batch", "delta",        "economic-event", "materialization", "packet",
      "query", "query-result", "snapshot",       "subscription"};
  for (size_t i = 0U; i < 9U; i++)
    (void)tw_dp_verify(kinds[i], data, size);
  return 0;
}
