import {Circle, Layout, makeScene2D} from '@motion-canvas/2d';
import {
  createRef, 
  tween, 
  waitFor,
  all,
  easeOutBack
} from '@motion-canvas/core';
import {createSignal, createEffect} from '@motion-canvas/core';

export default makeScene2D(function* (view) {
  // Configuration
  const TOTAL_CIRCLES = 6;
  const ANIMATION_DURATION = 3;
  const STAGGER_DELAY = 0.4; // Delay between each circle appearance
  
  // State
  const targetCount = createSignal(0);
  const circles = Array.from({ length: TOTAL_CIRCLES }, () => createRef<Circle>());
  
  // Layout container
  const container = createRef<Layout>();
  view.add(
    <Layout 
      ref={container} 
      width={view.width()} 
      height={view.height()} 
      gap={40}
      justifyContent={'center'} 
      alignItems={'center'} 
      layout
    />
  );

  // Wait for layout to initialize
  yield* waitFor(0.1);

  // Effect: Add/remove circles based on target count
  createEffect(() => {
    const currentCount = container().children().length;
    const target = targetCount();
    
    if (currentCount < target) {
      // Add missing circles
      for (let i = currentCount; i < target; i++) {
        container().add(
          <Circle 
            ref={circles[i]} 
            size={80} 
            fill={'#ff6b6b'} 
            scale={0} // Start invisible
          />
        );
      }
    } else if (currentCount > target) {
      // Remove excess circles
      const newChildren = container().children().slice(0, target);
      container().children(newChildren);
    }
  });

  // Animation: Staggered circle appearance
  const animateCircles = function* () {
    for (let i = 0; i < TOTAL_CIRCLES; i++) {
      // Update target count to trigger circle creation
      targetCount(i + 1);
      
      // Wait a moment for the circle to be created
      yield* waitFor(0.1);
      
      // Animate the new circle with a nice bounce effect
      if (circles[i]()) {
        yield* all(
          circles[i]().scale(1, 0.6, easeOutBack),
          circles[i]().rotation(360, 0.8)
        );
      }
      
      // Wait before creating the next circle
      yield* waitFor(STAGGER_DELAY);
    }
  };

  // Animation: Remove circles with staggered timing
  const removeCircles = function* () {
    for (let i = TOTAL_CIRCLES - 1; i >= 0; i--) {
      if (circles[i]()) {
        yield* circles[i]().scale(0, 0.3);
      }
      targetCount(i);
      yield* waitFor(0.2);
    }
  };

  // Main animation sequence
  yield* waitFor(0.5); // Initial pause
  yield* animateCircles(); // Add circles with stagger
  yield* waitFor(1); // Hold full state
  yield* removeCircles(); // Remove circles with stagger
  yield* waitFor(0.5); // Final pause
}); 