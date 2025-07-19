import {Circle, Layout, Rect, Node, makeScene2D, Txt, saturate, contrast} from '@motion-canvas/2d';
import {
  all,
  createRef,
  easeInExpo,
  easeInOutExpo,
  waitFor,
  waitUntil,
  ThreadGenerator,
  chain,
  createSignal,
  slideTransition,
  Direction,
  easeOutCirc,
  createEaseInOutBack,
  range,
} from '@motion-canvas/core';
import { InterpolationFunction } from '@motion-canvas/core';

export default makeScene2D(function* (view) {
  // How to build the layout rectangles
  const circles = range(4).map(() => createRef<Circle>());
  const radius = createSignal(100);
  const radius_outside_circle = createSignal(view.width()/2);
  const theta_position = createSignal(0);
  const directions = [Direction.Left, Direction.Right, Direction.Top, Direction.Bottom];
  const colours = ['red', 'green', 'blue', 'yellow'];
  const angles_start = [0, Math.PI/2, Math.PI, 3*Math.PI/2];
  const positions_during = range(4).map(i => ({
    x: () => radius_outside_circle() * Math.cos(angles_start[i] + theta_position()),
    y: () => radius_outside_circle() * Math.sin(angles_start[i] + theta_position())
  }));

  for (let i = 0; i < 4; i++) {
    view.add(
      <Circle 
        width={() => radius() * 2}
        height={() => radius() * 2} 
        fill={colours[i]} 
        ref={circles[i]} 
        x={positions_during[i].x} 
        y={positions_during[i].y} 
      />
    );
  }
  
  yield* all(
    radius_outside_circle(100, 6, easeInOutExpo).to(200, 1),
    theta_position(4 * Math.PI, 6, easeInOutExpo)
  );
}); 

function* outsideInPositioning(circle: Circle, direction: Direction, view_width: number, view_height: number): ThreadGenerator {
  let position_start_x = null;
  let position_start_y = null;
  let position_end_x = null;
  let position_end_y = null;
  console.log(`logging circle ${circle}`)
  if (direction == Direction.Left) {
    position_start_x = -view_width/2;
    position_start_y = 0;
    position_end_x = - circle.width();
    position_end_y = position_start_y;
  } else if (direction == Direction.Right) {
    position_start_x = view_width/2 
    position_start_y = 0
    position_end_x = circle.width();
    position_end_y = position_start_y;
  } else if (direction == Direction.Top) {
    position_start_x = 0
    position_start_y = -view_height/2;
    position_end_x = position_start_x;
    position_end_y = -circle.height();
  } else if (direction == Direction.Bottom) {
    position_start_x = 0
    position_start_y = view_height/2;
    position_end_x = position_start_x;
    position_end_y = circle.height();
  }
  console.log(`position_start_x: ${position_start_x}, position_start_y: ${position_start_y}, position_end_x: ${position_end_x}, position_end_y: ${position_end_y}`)
  yield* all(
    circle.position.x(position_start_x, 0, easeInOutExpo).to(position_end_x, 2, easeInOutExpo),
    circle.position.y(position_start_y, 0, easeInOutExpo).to(position_end_y, 2, easeInOutExpo),
  )
}

function acceleratingRotation(circle:Circle, direction: Direction){

}