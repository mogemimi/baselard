#include <vectormath/Vector2.h>
#include <cmath>

namespace vectormath {

float Vector2::Length() const
{
    return std::sqrt(x * x + y * y);
}

} // namespace vectormath
