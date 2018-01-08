#include <engine/TimeSourceApple.h>
#include <mach/mach_time.h>
#include <assert>

namespace Engine {

TimeSourceApple::TimeSourceApple()
{
    mach_timebase_info_data_t timeBase;
    mach_timebase_info(&timeBase);

#ifdef DEBUG
    assert(0 != timeBase.denom);
#endif
    double nanoSeconds = static_cast<double>(timeBase.numer) / timeBase.denom;

    constexpr double nanoScale = (1.0 / 1000000000LL);
    secondsPerTick = nanoScale * nanoSeconds;
}

TimePoint TimeSourceApple::Now() const
{
    auto currentSeconds = mach_absolute_time() * secondsPerTick;
    return TimePoint(Duration(currentSeconds));
}

} // namespace Engine
