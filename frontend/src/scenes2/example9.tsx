import {Circle, makeScene2D, Path, Rect} from '@motion-canvas/2d';
import {createRef} from '@motion-canvas/core';
import {Node} from '@motion-canvas/2d';
import {logMethods} from './utils';
import {all, waitFor} from '@motion-canvas/core';
import {Layout} from '@motion-canvas/2d';
export default makeScene2D(function* (view) {
  view.fill('black');
  const path = createRef<Circle>();
  const circle = createRef<Circle>();
  const circle2 = createRef<Circle>();
  const arrow = createRef<Path>();
  const pathBox = createRef<Rect>();
  const circleBox = createRef<Rect>();
  const arrowBox = createRef<Rect>();
  view.add(
    
    <Layout justifyContent={'center'} alignItems={'center'} gap = {50} direction={'row'}>
    <Layout>
      <Rect
        ref={pathBox}
        width={200}
        height={200}
        stroke={'#444'}
        lineWidth={1}
        opacity={0}
      />
      <Circle
        ref={path}
        lineWidth={4}
        stroke={'#e13238'}
        height={100}
        width={100}
        start={0}
        end={0}
        position={[0, 0]}
      />
    </Layout>
    <Layout>
      <Rect
        ref={arrowBox}
        width={220}
        height={40}
        stroke={'#444'}
        lineWidth={1}
        opacity={0}
      />
      <Path
        ref={arrow}
        lineWidth={4}
        stroke={'#e13238'}
        data="M -80 0 L 80 0 M 70 -8 L 80 0 L 70 8"
        start={0}
        end={0}
      />
    </Layout>
    <Layout>
      <Rect
        ref={circleBox}
        width={220}
        height={220}
        stroke={'#444'}
        lineWidth={1}
        opacity={0}
      />
      <Circle
        ref={circle2}
        size={180}
        stroke={'#e13238'}
        lineWidth={4}
        start={0}
        end={0}
      />
    </Layout>
    </Layout>
  );
  // yield* waitFor(1);
  // logMethods(circle(), 3);
  console.log('circle', circle().x.context.getter());
  yield* all(...[circle().end(1, 1), circle2().end(1, 1), pathBox().opacity(1, 0), arrowBox().opacity(1, 0), circleBox().opacity(1, 0)]);
  yield* arrow().end(1, 1);
  
  yield* circle().fill('#e13238', 1);
  yield* circle2().fill('#e13238', 1);
});