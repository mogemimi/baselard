#include "Vector2Test.h"
#include <vectormath/Vector2.h>
#include <cassert>

void TestVector2()
{
    vectormath::Vector2 a = {3.0f, 4.0f};
    vectormath::Vector2 b = {4.0f, 3.0f};
    assert(a.Length() == b.Length());
}
