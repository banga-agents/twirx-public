#include "dataplane.h"

#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

static bool read_file(const char *path, uint8_t **data, size_t *length) {
  FILE *file = fopen(path, "rb");
  if (file == NULL)
    return false;
  if (fseek(file, 0L, SEEK_END) != 0) {
    (void)fclose(file);
    return false;
  }
  const long size = ftell(file);
  if (size <= 0L ||
      (unsigned long)size > (unsigned long)TW_DP_MAX_DOCUMENT_BYTES ||
      fseek(file, 0L, SEEK_SET) != 0) {
    (void)fclose(file);
    return false;
  }
  *data = malloc((size_t)size);
  if (*data == NULL) {
    (void)fclose(file);
    return false;
  }
  *length = (size_t)size;
  const bool complete = fread(*data, 1U, *length, file) == *length;
  const bool closed = fclose(file) == 0;
  if (!complete || !closed) {
    free(*data);
    *data = NULL;
    return false;
  }
  return true;
}
int main(int argc, char **argv) {
  if (argc != 3) {
    (void)fprintf(stderr, "usage: tw-verify-data-plane-c KIND FILE\n");
    return 2;
  }
  uint8_t *data = NULL;
  size_t length = 0U;
  if (!read_file(argv[2], &data, &length))
    return 1;
  const bool valid = tw_dp_verify(argv[1], data, length);
  free(data);
  return valid ? 0 : 1;
}
