#include "stringify.h"
#include <stdio.h>
#include <string.h>

int main()
{
    char buf[100];
    stringify_int(42, buf, 100);

    const char* expected = "42";

    if (strncmp(buf, expected, 3) != 0) {
        printf("actual = %s\n", buf);
        printf("expected = %s\n", expected);
        printf("[Test] Failed.\n");
        return 1;
    }
    printf("[Test] Passed.\n");
    return 0;
}
