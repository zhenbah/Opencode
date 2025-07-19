import {Circle, Rect, makeScene2D} from '@motion-canvas/2d';
import {
  all,
  createRef,
  easeInExpo,
  easeInOutExpo,
  waitFor,
  waitUntil,
  ThreadGenerator,
  chain,
} from '@motion-canvas/core';

export default makeScene2D(function* (view) {

  console.log(`view min height: ${view.minHeight()}`);
  console.log(`view min width: ${view.minWidth()}`);
  console.log(`view width: ${view.width()}`);
  console.log(`view height: ${view.height()}`);
  const view_height = view.height();
  const view_width = view.width();
  const circle = createRef<Circle>();
    /* now tect is  a ref function ie you set it (mycircle) and the use it () */ 
  /* this create a new Circle instance with these props, calls circle(new_instance) to store the reference. 
  /* now whenever you do circle(), it fetches that object */
  const big_circle = createRef<Circle>();
  const big_circle_obj : Circle = <Circle ref={big_circle} width={1100} height={1100} />
  const circle3_parent = createRef<Circle>();
  const circle4_child = createRef<Circle>();
  const circle3_obj: Circle = <Circle ref={circle3_parent} width={view_width / 6} height={view_height / 6} />
  const circle4_obj : Circle = <Circle ref={circle4_child} width={view_width / 10} height={view_width / 10} />
  big_circle_obj.position.x(0);
  big_circle_obj.position.y(0);
  big_circle_obj.fill('blue');
  big_circle_obj.opacity(0.3);

  circle3_obj.position([view_width / 4, -view_height / 4])
  circle3_obj.fill('red');
  circle3_obj.opacity(0.3);

  circle3_obj.add(circle4_obj);
  circle4_obj.position([0,0]);
  circle4_obj.fill('green');
  circle4_obj.opacity(0.8);

  view.add(
    [
      big_circle_obj,
      circle3_obj]
    );
    
    
    let circles_refs= Array.from({length: 10}, () => createRef<Circle>());
    let circles : Circle[] = circles_refs.map(ref => <Circle ref={ref} width={20} height={20} />) as Circle[];
    for (let i = 0; i < circles.length; i++){
      circles[i].fill('blue');
      circles[i].position.x(i * 40);
      circles[i].position.y(i * 40);
    }
    
    view.add(circles);  
    // console.log(view.children);
    console.log(circle4_obj.localToWorld()); 
    // expecting this to be not [0,0] but the [width/4, - height/4]


  yield* all(
    ...circles_refs.map(ref => randomColor(ref())),
    ...circles_refs.map(ref => rotationCircle(ref()))
  )
});



function* randomColor(circle: Circle): ThreadGenerator {
  const colors = ['blue', 'red', 'green', 'yellow', 'purple', 'orange', 'pink'];
  let random_color = colors[Math.floor(Math.random() * colors.length)];
  yield* circle.fill(random_color, 0.5);
}
function* rotationCircle(circle:Circle) : ThreadGenerator { 
  let new_positions = []
  let num_increments = 30;
  for (let i = 0; i < num_increments; i++){
    let new_theta = i * 2 * Math.PI / num_increments;
    let x = circle.position.x(); 
    let y = circle.position.y();
    let r = Math.sqrt(x*x + y*y);
    // = rcos(theta)
    let new_x = r * Math.cos(new_theta);
    let new_y = r * Math.sin(new_theta);
    new_positions.push(new_x, new_y);
  }
  // yield* circle.position.x(new_positions[0], 10, easeInOutExpo);
  // yield* circle.position.y(new_positions[1], 10, easeInOutExpo);
  for (let i = 0; i < new_positions.length; i+=2){
    yield* all(
      circle.position.x(new_positions[i], 0.5, easeInOutExpo), 
      circle.position.y(new_positions[i+1], 0.5, easeInOutExpo),
      randomColor(circle)
    )
  }
}