import {makeScene2D, Rect, Node, Txt} from '@motion-canvas/2d';
import {createRef} from '@motion-canvas/core';

export default makeScene2D(function* (view) {
  // Refs for components
  const background = createRef<Rect>();
  const container = createRef<Node>();
  const greeting = createRef<Txt>();

  // Build scene graph
  view.add(
    <Rect
      ref={background}
      width={() => view.width()}
      height={() => view.height()}
      fill={() => '#000000'}
    >
      <Node
        ref={container}
        x={() => 0}
        y={() => 0}
      >
        <Txt
          ref={greeting}
          x={() => 0}
          y={() => 0}
          text={() => 'Hello there'}
          fontSize={() => 64}
          fill={() => '#ffffff'}
        />
      </Node>
    </Rect>
  );
});