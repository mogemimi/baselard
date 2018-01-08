#pragma once

#include <engine/TimePoint.h>

namespace Engine {

class TimeSourceWindows {
public:
    TimeSourceWindows();

    TimePoint Now() const;

private:
    double secondsPerTick;
};

} // namespace Engine
