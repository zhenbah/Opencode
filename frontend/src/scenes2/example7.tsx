import {Circle, makeScene2D} from '@motion-canvas/2d';
import {
  createRef, 
  tween, 
  waitFor
} from '@motion-canvas/core';
import {Layout} from '@motion-canvas/2d';
import {createSignal} from '@motion-canvas/core';
import {createEffect} from '@motion-canvas/core';
import {map} from '@motion-canvas/core';

export default makeScene2D(function* (view) {
  const layout = createRef<Layout>();
  const count = createSignal(0);
  const max_circles = 10;
  const circles = Array.from({ length: max_circles }, () => createRef<Circle>());

  view.add(
    <Layout ref={layout} x={0} y={0} width={view.width()} height={view.height()} gap = {30} justifyContent={'center'} alignItems={'center'} layout>
    </Layout>
  );

  // Wait a frame to let the layout initialize
  yield* waitFor(0);

  const num_circles_current = () => layout().children().length;
  
  // Simple effect - just add/remove circles
  const effect = createEffect(() => {
    console.log(`Effect triggered! count is ${count()}`);
    
    if (num_circles_current() < count()) {
      const circleIndex = num_circles_current();
      layout().add(
        <Circle ref={circles[circleIndex]} width={100} height={100} fill={'red'} />
      );
      // Start new circles at scale 0
      circles[circleIndex]().scale(0);
    } else if (num_circles_current() > count()) {
      layout().children(layout().children().slice(0, count()));
    }
  });

  yield* tween(max_circles, (value) => { 
    let old_count = count();
    const value_starts = Array.from({ length: max_circles }, (_, k) => k / max_circles);
    count(Math.ceil(value * max_circles));
    console.log(`value_starts ${value_starts}`);
    for (let i = 0; i < count(); i++) {
        let time_left = max_circles - i ;
        let coef = 1/ time_left;
        console.log(`i = ${i} coef = ${coef} value = ${value} -> ${coef*(value-value_starts[i])*max_circles}`)
        if(circles[i]()){
            circles[i]().scale(map(0,1,coef*(value-value_starts[i])*max_circles ))
            if (old_count< count()) { console.log(`new circle created ! `)}
    }}
  });
  
});
