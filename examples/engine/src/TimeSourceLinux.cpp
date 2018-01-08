#include <engine/TimeSourceLinux.h>
#include <mach/mach_time.h>
#include <cassert>

namespace Engine {

TimePoint TimeSourceLinux::Now() const
{
    struct timespec now;
#ifdef DEBUG
    assert(0 == clock_gettime(CLOCK_MONOTONIC, &now));
#endif

    constexpr double nanoScale = (1.0 / 1000000000LL);

    return TimePoint { Duration {
        static_cast<double>(now.tv_sec) +
        static_cast<double>(now.tv_nsec) * nanoScale}};
}

} // namespace Engine
