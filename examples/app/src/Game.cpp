#include <app/Game.h>
#include <vectormath/Vector2.h>
#include <iostream>

namespace App {

int Game::Run()
{
    vectormath::Vector2 pos = {3.0f, 4.0f};
    std::cout << pos.Length() << std::endl;
    std::cout << "Hello." << std::endl;
    return 0;
}

} // namespace App
