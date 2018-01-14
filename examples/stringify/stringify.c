#include "stringify.h"
#include <stdio.h>

int stringify_int(int c, char* s, size_t n)
{
    return snprintf(s, n, "%d", c);
}
