#pragma once

#include <chrono>

namespace Engine {

using Duration = std::chrono::duration<double>;

class GameClock;
using TimePoint = std::chrono::time_point<GameClock, Duration>;

} // namespace Engine
